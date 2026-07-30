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

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

// bddWebAuthParams builds Parameter.Params for BDD web auth tokens.
// Scopes must satisfy route.RequireServiceScope for /service/web (pipeline:read/run
// or admin:*).
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

// bddWebScopesUser passes /service/web Authorize (GET and POST) but not admin:*.
// pipeline:run also satisfies pipeline:read for read routes.
func bddWebScopesUser() []string {
	return []string{auth.ScopePipelineRun}
}

// bddNoopParameterSet is a no-op ParameterSet for legacy BDD Adapter.ParameterSet stubs.
func bddNoopParameterSet(_ context.Context, _ string, _ types.KV, _ time.Time) error {
	return nil
}

// bddSeedAccessToken persists a hashed access token for Authorize / authenticateWeb.
// After the store facade split, route lookup uses ModuleDataStore (not Adapter.ParameterGet).
func bddSeedAccessToken(rawToken, uid string, scopes []string) {
	Expect(store.NewModuleDataStore(EntClient).ParameterSet(
		context.Background(),
		auth.HashToken(rawToken),
		types.KV(bddWebAuthParams(uid, scopes)),
		time.Now().Add(time.Hour),
	)).To(Succeed())
}

// bddEnsureWebAccount creates the first admin web account when the DB has none
// (login BDD specs need a real row; InitForE2E does not run Bootstrap migration).
func bddEnsureWebAccount(username, password string) {
	ws := store.NewWebAccountStore(EntClient)
	n, err := ws.Count(context.Background())
	Expect(err).NotTo(HaveOccurred())
	if n > 0 {
		return
	}
	hash, err := webauth.HashPassword(password)
	Expect(err).NotTo(HaveOccurred())
	_, err = ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     username,
		PasswordHash: hash,
	})
	Expect(err).NotTo(HaveOccurred())
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
