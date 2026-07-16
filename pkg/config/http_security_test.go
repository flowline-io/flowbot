package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSendHSTS(t *testing.T) {
	t.Parallel()
	trueVal := true
	falseVal := false
	tests := []struct {
		name string
		cfg  Type
		want bool
	}{
		{
			name: "tls_behind_proxy alone enables HSTS",
			cfg:  Type{HTTP: HTTPConfig{TLSBehindProxy: true}},
			want: true,
		},
		{
			name: "no tls and no web module disables HSTS",
			cfg:  Type{},
			want: false,
		},
		{
			name: "web cookie_secure omitted defaults to HSTS on",
			cfg: Type{
				Modules: []map[string]any{
					{"name": "web", "auth": map[string]any{}},
				},
			},
			want: true,
		},
		{
			name: "web cookie_secure true enables HSTS",
			cfg: Type{
				Modules: []map[string]any{
					{"name": "web", "auth": map[string]any{"cookie_secure": trueVal}},
				},
			},
			want: true,
		},
		{
			name: "web cookie_secure false disables HSTS without tls_behind_proxy",
			cfg: Type{
				Modules: []map[string]any{
					{"name": "web", "auth": map[string]any{"cookie_secure": falseVal}},
				},
			},
			want: false,
		},
		{
			name: "tls_behind_proxy overrides cookie_secure false",
			cfg: Type{
				HTTP: HTTPConfig{TLSBehindProxy: true},
				Modules: []map[string]any{
					{"name": "web", "auth": map[string]any{"cookie_secure": falseVal}},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.cfg.ShouldSendHSTS())
		})
	}
}
