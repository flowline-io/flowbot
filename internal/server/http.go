/******************************************************************************
 *
 *  Description :
 *
 *  Web server initialization and shutdown.
 *
 *****************************************************************************/

package server

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/version"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/flowline-io/flowbot/pkg/logs"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/queue"
)

func listenAndServe(addr string, mux *http.ServeMux, tlfConf *tls.Config, stop <-chan bool) error {
	globals.shuttingDown = false

	httpdone := make(chan bool)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		MaxHeaderBytes:    1 << 14,
	}

	server.TLSConfig = tlfConf

	go func() {
		var err error
		if server.TLSConfig != nil {
			// If port is not specified, use default https port (443),
			// otherwise it will default to 80
			if addr == "" {
				addr = ":https"
			}

			if globals.tlsRedirectHTTP != "" {
				// Serving redirects from a unix socket or to a unix socket makes no sense.
				if utils.IsUnixAddr(globals.tlsRedirectHTTP) || utils.IsUnixAddr(addr) {
					err = errors.New("HTTP to HTTPS redirect: unix sockets not supported")
				} else {
					logs.Info.Printf("Redirecting connections from HTTP at [%s] to HTTPS at [%s]",
						globals.tlsRedirectHTTP, addr)

					// This is a second HTTP server listenning on a different port.
					go func() {
						if err := http.ListenAndServe(globals.tlsRedirectHTTP, tlsRedirect(addr)); err != nil && err != http.ErrServerClosed {
							logs.Info.Println("HTTP redirect failed:", err)
						}
					}()
				}
			}

			if err == nil {
				logs.Info.Printf("Listening for client HTTPS connections on [%s]", addr)
				var lis net.Listener
				lis, err = utils.NetListener(addr)
				if err == nil {
					err = server.ServeTLS(lis, "", "")
				}
			}
		} else {
			logs.Info.Printf("Listening for client HTTP connections on [%s]", addr)
			var lis net.Listener
			lis, err = utils.NetListener(addr)
			if err == nil {
				err = server.Serve(lis)
			}
		}

		if err != nil {
			if globals.shuttingDown {
				logs.Info.Println("HTTP server: stopped")
			} else {
				logs.Err.Println("HTTP server: failed", err)
			}
		}
		httpdone <- true
	}()

	// Wait for either a termination signal or an error
Loop:
	for {
		select {
		case <-stop:
			// Flip the flag that we are terminating and close the Accept-ing socket, so no new connections are possible.
			globals.shuttingDown = true
			// Give server 2 seconds to shut down.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := server.Shutdown(ctx); err != nil {
				// failure/timeout shutting down the server gracefully
				logs.Err.Println("HTTP server failed to terminate gracefully", err)
			}

			// While the server shuts down, termianate all sessions.
			globals.sessionStore.Shutdown()

			// Wait for http server to stop Accept()-ing connections.
			<-httpdone
			cancel()

			// Stop publishing statistics.
			statsShutdown()

			// Shutdown the hub. The hub will shutdown topics.
			hubdone := make(chan bool)
			globals.hub.shutdown <- hubdone

			// Wait for the hub to finish.
			<-hubdone

			// Shutdown Extra
			globals.crawler.Shutdown()
			globals.worker.Shutdown()
			globals.scheduler.Shutdown()
			globals.manager.Shutdown()
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}
			event.Shutdown()
			queue.Shutdown()
			cache.Shutdown()

			break Loop

		case <-httpdone:
			break Loop
		}
	}
	return nil
}

func signalHandler() <-chan bool {
	stop := make(chan bool)

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		// Wait for a signal. Don't care which signal it is
		sig := <-signchan
		logs.Info.Printf("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}

// Wrapper for http.Handler which optionally adds a Strict-Transport-Security to the response.
func hstsHandler(handler http.Handler) http.Handler {
	if globals.tlsStrictMaxAge != "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age="+globals.tlsStrictMaxAge)
			handler.ServeHTTP(w, r)
		})
	}
	return handler
}

// The following code is used to intercept HTTP errors so they can be wrapped into json.

// Wrapper around http.ResponseWriter which detects status set to 400+ and replaces
// default error message with a custom one.
type errorResponseWriter struct {
	status int
	http.ResponseWriter
}

func (w *errorResponseWriter) WriteHeader(status int) {
	if status >= http.StatusBadRequest {
		// charset=utf-8 is the default. No need to write it explicitly
		// Must set all the headers before calling super.WriteHeader()
		w.ResponseWriter.Header().Set("Content-Type", "application/json")
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *errorResponseWriter) Write(p []byte) (n int, err error) {
	if w.status >= http.StatusBadRequest {
		p, _ = json.Marshal(
			&ServerComMessage{
				Ctrl: &MsgServerCtrl{
					Timestamp: time.Now().UTC().Round(time.Millisecond),
					Code:      w.status,
					Text:      http.StatusText(w.status),
				},
			})
	}
	return w.ResponseWriter.Write(p)
}

// Handler which deploys errorResponseWriter
func httpErrorHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(&errorResponseWriter{0, w}, r)
		})
}

// Custom 404 response.
func serve404(wrt http.ResponseWriter, req *http.Request) {
	wrt.Header().Set("Content-Type", "application/json; charset=utf-8")
	wrt.WriteHeader(http.StatusNotFound)
	json.NewEncoder(wrt).Encode(
		&ServerComMessage{
			Ctrl: &MsgServerCtrl{
				Timestamp: time.Now().UTC().Round(time.Millisecond),
				Code:      http.StatusNotFound,
				Text:      "not found",
			},
		})
}

// Redirect HTTP requests to HTTPS
func tlsRedirect(toPort string) http.HandlerFunc {
	if toPort == ":443" || toPort == ":https" {
		toPort = ""
	} else if toPort != "" && toPort[:1] == ":" {
		// Strip leading colon. JoinHostPort will add it back.
		toPort = toPort[1:]
	}

	return func(wrt http.ResponseWriter, req *http.Request) {
		host, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			// If SplitHostPort has failed assume it's because :port part is missing.
			host = req.Host
		}

		target, _ := url.ParseRequestURI(req.RequestURI)
		target.Scheme = "https"

		// Ensure valid redirect target.
		if toPort != "" {
			// Replace the port number.
			target.Host = net.JoinHostPort(host, toPort)
		} else {
			target.Host = host
		}

		if target.Path == "" {
			target.Path = "/"
		}

		http.Redirect(wrt, req, target.String(), http.StatusTemporaryRedirect)
	}
}

// Wrapper for http.Handler which optionally adds a Cache-Control header to the response
func cacheControlHandler(maxAge int, handler http.Handler) http.Handler {
	if maxAge > 0 {
		strMaxAge := strconv.Itoa(maxAge)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "must-revalidate, public, max-age="+strMaxAge)
			handler.ServeHTTP(w, r)
		})
	}
	return handler
}

// Get API key from an HTTP request.
func getAPIKey(req *http.Request) string {
	// Check header.
	apikey := req.Header.Get("X-Tinode-APIKey")
	if apikey != "" {
		return apikey
	}

	// Check URL query parameters.
	apikey = req.URL.Query().Get("apikey")
	if apikey != "" {
		return apikey
	}

	// Check form values.
	apikey = req.FormValue("apikey")
	if apikey != "" {
		return apikey
	}

	// Check cookies.
	if c, err := req.Cookie("apikey"); err == nil {
		apikey = c.Value
	}

	return apikey
}

// Extracts authorization credentials from an HTTP request.
// Returns authentication method and secret.
func getHttpAuth(req *http.Request) (method, secret string) {
	// Check X-Tinode-Auth header.
	if parts := strings.Split(req.Header.Get("X-Tinode-Auth"), " "); len(parts) == 2 {
		method, secret = parts[0], parts[1]
		return
	}

	// Check canonical Authorization header.
	if parts := strings.Split(req.Header.Get("Authorization"), " "); len(parts) == 2 {
		method, secret = parts[0], parts[1]
		return
	}

	// Check URL query parameters.
	if method = req.URL.Query().Get("auth"); method != "" {
		// Get the auth secret.
		secret = req.URL.Query().Get("secret")
		// Convert base64 URL-encoding to standard encoding.
		secret = strings.NewReplacer("-", "+", "_", "/").Replace(secret)
		return
	}

	// Check form values.
	if method = req.FormValue("auth"); method != "" {
		return method, req.FormValue("secret")
	}

	// Check cookies as the last resort.
	if mcookie, err := req.Cookie("auth"); err == nil {
		if scookie, err := req.Cookie("secret"); err == nil {
			method, secret = mcookie.Value, scookie.Value
		}
	}

	return
}

// Authenticate non-websocket HTTP request
func authHttpRequest(req *http.Request) (types.Uid, []byte, error) {
	var uid types.Uid
	if authMethod, secret := getHttpAuth(req); authMethod != "" {
		decodedSecret := make([]byte, base64.StdEncoding.DecodedLen(len(secret)))
		_, err := base64.StdEncoding.Decode(decodedSecret, []byte(secret))
		if err != nil {
			logs.Info.Println("media: invalid auth secret", authMethod, "'"+secret+"'")
			return uid, nil, types.ErrMalformed
		}
	} else {
		// Find the session, make sure it's appropriately authenticated.
		sess := globals.sessionStore.Get(req.FormValue("sid"))
		if sess != nil {
			uid = sess.uid
		}
	}
	return uid, nil, nil
}

// debugSession is session debug info.
type debugSession struct {
	RemoteAddr string   `json:"remote_addr,omitempty"`
	Ua         string   `json:"ua,omitempty"`
	Uid        string   `json:"uid,omitempty"`
	Sid        string   `json:"sid,omitempty"`
	Clnode     string   `json:"clnode,omitempty"`
	Subs       []string `json:"subs,omitempty"`
}

// debugTopic is a topic debug info.
type debugTopic struct {
	Topic    string   `json:"topic,omitempty"`
	Xorig    string   `json:"xorig,omitempty"`
	IsProxy  bool     `json:"is_proxy,omitempty"`
	PerUser  []string `json:"per_user,omitempty"`
	PerSubs  []string `json:"per_subs,omitempty"`
	Sessions []string `json:"sessions,omitempty"`
}

// debugCachedUser is a user cache entry debug info.
type debugCachedUser struct {
	Uid    string `json:"uid,omitempty"`
	Unread int    `json:"unread,omitempty"`
	Topics int    `json:"topics,omitempty"`
}

// debugDump is server internal state dump for debugging.
type debugDump struct {
	Version   string            `json:"server_version,omitempty"`
	Build     string            `json:"build_id,omitempty"`
	Timestamp time.Time         `json:"ts,omitempty"`
	Sessions  []debugSession    `json:"sessions,omitempty"`
	Topics    []debugTopic      `json:"topics,omitempty"`
	UserCache []debugCachedUser `json:"user_cache,omitempty"`
}

func serveStatus(wrt http.ResponseWriter, req *http.Request) {
	wrt.Header().Set("Content-Type", "application/json")

	result := &debugDump{
		Version:   version.CurrentVersion,
		Build:     version.Buildstamp,
		Timestamp: types.TimeNow(),
		Sessions:  make([]debugSession, 0, len(globals.sessionStore.sessCache)),
		Topics:    make([]debugTopic, 0, 10),
		UserCache: make([]debugCachedUser, 0, 10),
	}
	// Sessions.
	globals.sessionStore.Range(func(sid string, s *Session) bool {
		keys := make([]string, 0, len(s.subs))
		for tn := range s.subs {
			keys = append(keys, tn)
		}
		sort.Strings(keys)
		var clnode string
		result.Sessions = append(result.Sessions, debugSession{
			RemoteAddr: s.remoteAddr,
			Ua:         s.userAgent,
			Uid:        s.uid.String(),
			Sid:        sid,
			Clnode:     clnode,
			Subs:       keys,
		})
		return true
	})

	json.NewEncoder(wrt).Encode(result)
}
