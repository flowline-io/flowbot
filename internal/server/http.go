package server

import (
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/flowline-io/flowbot/pkg/version"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

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

func serveStatus(wrt http.ResponseWriter, _ *http.Request) {
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

	_ = json.NewEncoder(wrt).Encode(result)
}
