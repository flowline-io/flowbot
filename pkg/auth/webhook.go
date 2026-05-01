package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const DefaultWebhookMaxSkew = 5 * time.Minute

func WebhookBodyHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func WebhookSignaturePayload(method, path string, timestamp time.Time, body []byte) string {
	return strings.ToUpper(method) + "\n" + path + "\n" + fmt.Sprintf("%d", timestamp.Unix()) + "\n" + WebhookBodyHash(body)
}

func SignWebhook(secret, method, path string, timestamp time.Time, body []byte) string {
	payload := WebhookSignaturePayload(method, path, timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyWebhookSignature(secret, method, path string, timestamp time.Time, body []byte, signature string, now time.Time, maxSkew time.Duration) bool {
	if secret == "" || signature == "" || timestamp.IsZero() {
		return false
	}
	if maxSkew <= 0 {
		maxSkew = DefaultWebhookMaxSkew
	}
	delta := now.Sub(timestamp)
	if delta < -maxSkew || delta > maxSkew {
		return false
	}
	expected := SignWebhook(secret, method, path, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
