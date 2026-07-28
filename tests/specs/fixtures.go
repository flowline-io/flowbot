//go:build integration
// +build integration

package specs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

// bddWebAuthParams builds Parameter.Params for BDD web auth stubs.
// Scopes must satisfy route.RequireServiceScope for /service/web (pipeline:read
// or admin:*). Stubs that override ParameterGet must also implement ParameterSet
// (use bddNoopParameterSet) — Authorize updates last_used_at after token lookup.
func bddWebAuthParams(uid string, scopes []string) map[string]any {
	return map[string]any{
		"uid":    uid,
		"topic":  "test",
		"kind":   webauth.KindFull,
		"scopes": scopes,
	}
}

// bddWebScopesAdmin passes /service/web Authorize and admin UI gates.
func bddWebScopesAdmin() []string {
	return []string{auth.ScopeAdmin}
}

// bddWebScopesUser passes /service/web Authorize but not admin:*.
func bddWebScopesUser() []string {
	return []string{auth.ScopePipelineRead}
}

// bddNoopParameterSet is a no-op ParameterSet for BDD stubs with a nil embedded Adapter.
func bddNoopParameterSet(_ context.Context, _ string, _ types.KV, _ time.Time) error {
	return nil
}

// MakeRequest creates an HTTP request for testing.
func MakeRequest(method, path string, body []byte) *http.Request {
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}
	url := "http://localhost" + path
	req, err := http.NewRequest(method, url, bodyReader)
	Expect(err).NotTo(HaveOccurred())
	return req
}

// JSONRequest creates a JSON HTTP request for testing.
func JSONRequest(method, path string, body []byte) *http.Request {
	req := MakeRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ReadBody reads and returns the response body.
func ReadBody(resp *http.Response) []byte {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	return body
}
