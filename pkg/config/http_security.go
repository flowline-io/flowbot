package config

import (
	"github.com/bytedance/sonic"
)

// webModuleAuthProbe is used to decode modules.web.auth.cookie_secure for HSTS.
type webModuleAuthProbe struct {
	CookieSecure *bool `json:"cookie_secure"`
}

// webModuleProbe is used to locate the web module entry in modules config.
type webModuleProbe struct {
	Name string             `json:"name"`
	Auth webModuleAuthProbe `json:"auth"`
}

// ShouldSendHSTS reports whether Strict-Transport-Security should be set.
// True when http.tls_behind_proxy is set, or modules.web.auth.cookie_secure
// is enabled (default true when the web module omits the field).
func (t *Type) ShouldSendHSTS() bool {
	if t == nil {
		return false
	}
	if t.HTTP.TLSBehindProxy {
		return true
	}
	return t.webCookieSecureEnabled()
}

// webCookieSecureEnabled reads modules.web.auth.cookie_secure.
// Matches the web module default: true when the field is omitted.
// Returns false when the web module is absent or cookie_secure is explicitly false.
func (t *Type) webCookieSecureEnabled() bool {
	if t == nil || t.Modules == nil {
		return false
	}
	raw, err := sonic.Marshal(t.Modules)
	if err != nil {
		return false
	}
	var mods []webModuleProbe
	if err := sonic.Unmarshal(raw, &mods); err != nil {
		return false
	}
	for _, m := range mods {
		if m.Name != "web" {
			continue
		}
		if m.Auth.CookieSecure == nil {
			return true
		}
		return *m.Auth.CookieSecure
	}
	return false
}
