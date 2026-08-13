package functions

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AuthenticateCall validates a function HTTP call using metadata token and/or HMAC.
// Token auth accepts headerToken (X-Webhook-Token) or queryToken. HMAC uses
// X-Hub-Signature-256 style "sha256=<hex>" signatures over the raw body.
func AuthenticateCall(meta *Metadata, headerToken, queryToken, hmacSig string, body []byte) bool {
	if meta == nil {
		return false
	}
	token := strings.TrimSpace(meta.HTTP.Auth.Token)
	hmacSecret := strings.TrimSpace(meta.HTTP.Auth.HMACSecret)
	if token == "" && hmacSecret == "" {
		return false
	}
	if token != "" {
		provided := strings.TrimSpace(headerToken)
		if provided == "" {
			provided = strings.TrimSpace(queryToken)
		}
		if provided != "" && hmac.Equal([]byte(provided), []byte(token)) {
			return true
		}
	}
	if hmacSecret != "" && VerifyHMACSHA256(hmacSecret, body, hmacSig) {
		return true
	}
	return false
}

// VerifyHMACSHA256 verifies an HMAC-SHA256 signature against the body.
// Accepts signatures of the form "sha256=<hex>" (case-insensitive prefix).
func VerifyHMACSHA256(secret string, body []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(strings.ToLower(signature), prefix) {
		return false
	}
	expectedHex := strings.TrimPrefix(strings.ToLower(signature), prefix)
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	actual := mac.Sum(nil)
	return hmac.Equal(actual, expected)
}
