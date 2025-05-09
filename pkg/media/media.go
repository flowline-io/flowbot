// Package media defines an interface which must be implemented by media upload/download handlers.
package media

import (
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ReadSeekCloser must be implemented by the media being downloaded.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// Handler is an interface which must be implemented by media handlers (uploader-downloader).
type Handler interface {
	// Init initializes the media upload handler.
	Init(jsconf string) error

	// Headers checks if the handler wants to provide additional HTTP headers for the request.
	// It could be CORS headers, redirect to serve files from another URL, cache-control headers.
	// It returns headers as a map, HTTP status code to stop processing or 0 to continue, error.
	Headers(req *http.Request, serve bool) (http.Header, int, error)

	// Upload processes request for file upload. Returns file URL, file size, error.
	Upload(fdef *types.FileDef, file io.ReadSeeker) (string, int64, error)

	// Download processes request for file download.
	Download(url string) (*types.FileDef, ReadSeekCloser, error)

	// Delete deletes file from storage.
	Delete(locations []string) error

	// GetIdFromUrl extracts file ID from download URL.
	GetIdFromUrl(url string) types.Uid
}

var fileNamePattern = regexp.MustCompile(`^[-_A-Za-z0-9]+`)

// GetIdFromUrl is a helper method for extracting file ID from a URL.
func GetIdFromUrl(url, serveUrl string) types.Uid {
	dir, fname := path.Split(path.Clean(url))

	if dir != "" && dir != serveUrl {
		return types.ZeroUid
	}

	return types.Uid(fileNamePattern.FindString(fname))
}

// matchCORSOrigin compares origin from the HTTP request to a list of allowed origins.
func matchCORSOrigin(allowed []string, origin string) string {
	if origin == "" {
		// Request has no Origin header.
		return ""
	}

	if len(allowed) == 0 {
		// Not configured
		return ""
	}

	if allowed[0] == "*" {
		return "*"
	}

	origin = strings.ToLower(origin)
	for _, val := range allowed {
		if strings.ToLower(val) == origin {
			return origin
		}
	}

	return ""
}

func matchCORSMethod(allowMethods []string, method string) bool {
	if method == "" {
		// Request has no Method header.
		return false
	}

	method = strings.ToUpper(method)
	for _, mm := range allowMethods {
		if strings.ToUpper(mm) == method {
			return true
		}
	}

	return false
}

// CORSHandler is the default preflight OPTIONS processor for use by media handlers.
func CORSHandler(req *http.Request, allowedOrigins []string, serve bool) (http.Header, int) {
	if req.Method != http.MethodOptions {
		// Not an OPTIONS request. No special handling for all other requests.
		return nil, 0
	}

	var allowMethods []string
	if serve {
		allowMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	} else {
		allowMethods = []string{http.MethodPost, http.MethodPut, http.MethodHead, http.MethodOptions}
	}

	headers := map[string][]string{
		// Always add Vary because of possible intermediate caches.
		"Vary":                             {"Origin", "Access-Control-Request-Method"},
		"Access-Control-Allow-Headers":     {"*"},
		"Access-Control-Max-Age":           {"86400"},
		"Access-Control-Allow-Credentials": {"true"},
		"Access-Control-Allow-Methods":     {strings.Join(allowMethods, ", ")},
	}

	if !matchCORSMethod(allowMethods, req.Header.Get("Access-Control-Request-Method")) {
		// CORS policy does not allow this method.
		return headers, http.StatusNoContent
	}

	allowedOrigin := matchCORSOrigin(allowedOrigins, req.Header.Get("Origin"))
	if allowedOrigin == "" {
		// CORS policy does not match the origin.
		return headers, http.StatusNoContent
	}

	headers["Access-Control-Allow-Origin"] = []string{allowedOrigin}

	return headers, http.StatusNoContent
}
