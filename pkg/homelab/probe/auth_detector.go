package probe

import (
	"net/http"
	"strings"

	"github.com/flowline-io/flowbot/pkg/homelab"
)

// AuthDetector analyses HTTP responses to determine the authentication
// mechanism used by an API endpoint.
type AuthDetector struct{}

// Detect analyses the HTTP response to determine the auth type, header, and
// prefix. Returns an AuthInfo with AuthUnknown when the mechanism cannot be
// determined; returns nil only for a nil response.
func (d *AuthDetector) Detect(resp *http.Response) *homelab.AuthInfo {
	if resp == nil {
		return nil
	}

	status := resp.StatusCode
	wwwAuth := resp.Header.Get("WWW-Authenticate")

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		if strings.HasPrefix(wwwAuth, "Bearer") {
			return d.parseBearerAuth(resp)
		}
		if strings.HasPrefix(wwwAuth, "Basic") {
			return &homelab.AuthInfo{
				Type:   homelab.AuthBasic,
				Header: "Authorization",
				Prefix: "Basic",
			}
		}
		return d.detectAPIKey(resp)

	case status >= 200 && status < 300:
		return &homelab.AuthInfo{Type: homelab.AuthNone}

	default:
		return &homelab.AuthInfo{Type: homelab.AuthUnknown}
	}
}

func (d *AuthDetector) parseBearerAuth(resp *http.Response) *homelab.AuthInfo {
	auth := &homelab.AuthInfo{
		Type:   homelab.AuthOAuth2,
		Header: "Authorization",
		Prefix: "Bearer",
	}
	// Check for OIDC-specific headers in the response.
	if resp.Header.Get("X-OIDC-Issuer") != "" {
		auth.Type = homelab.AuthOIDC
	}
	return auth
}

func (d *AuthDetector) detectAPIKey(resp *http.Response) *homelab.AuthInfo {
	// When a server returns 401/403 without WWW-Authenticate, it often uses
	// an API key in a custom header or query parameter. We cannot determine
	// the exact header name from the response alone.
	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if wwwAuth == "" {
		return &homelab.AuthInfo{
			Type: homelab.AuthAPIToken,
		}
	}
	return &homelab.AuthInfo{Type: homelab.AuthUnknown}
}


