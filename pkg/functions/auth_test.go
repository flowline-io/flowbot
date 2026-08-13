package functions_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/functions"
)

func TestAuthenticateCall(t *testing.T) {
	t.Parallel()
	body := []byte(`{"x":1}`)
	mac := hmac.New(sha256.New, []byte("hmac-secret"))
	_, _ = mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name        string
		meta        *functions.Metadata
		headerToken string
		queryToken  string
		hmacSig     string
		want        bool
	}{
		{
			name: "token header match",
			meta: &functions.Metadata{HTTP: functions.HTTPConfig{Auth: functions.HTTPAuth{Token: "secret"}}},
			headerToken: "secret",
			want:        true,
		},
		{
			name: "token query match",
			meta: &functions.Metadata{HTTP: functions.HTTPConfig{Auth: functions.HTTPAuth{Token: "secret"}}},
			queryToken:  "secret",
			want:        true,
		},
		{
			name: "token mismatch",
			meta: &functions.Metadata{HTTP: functions.HTTPConfig{Auth: functions.HTTPAuth{Token: "secret"}}},
			headerToken: "wrong",
			want:        false,
		},
		{
			name: "hmac match",
			meta: &functions.Metadata{HTTP: functions.HTTPConfig{Auth: functions.HTTPAuth{HMACSecret: "hmac-secret"}}},
			hmacSig:     sig,
			want:        true,
		},
		{
			name: "hmac mismatch",
			meta: &functions.Metadata{HTTP: functions.HTTPConfig{Auth: functions.HTTPAuth{HMACSecret: "hmac-secret"}}},
			hmacSig:     "sha256=deadbeef",
			want:        false,
		},
		{
			name: "nil metadata",
			meta: nil,
			want: false,
		},
		{
			name: "no secrets configured",
			meta: &functions.Metadata{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := functions.AuthenticateCall(tt.meta, tt.headerToken, tt.queryToken, tt.hmacSig, body)
			assert.Equal(t, tt.want, got)
		})
	}
}
