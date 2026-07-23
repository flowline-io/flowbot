package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// SignedPathPrefix is the HTTP path prefix for HMAC-signed media downloads (FS backend).
	SignedPathPrefix = "/chatagent/media/s/"
)

// BuildSignedURL builds an absolute HMAC-signed GET URL for a file id.
func BuildSignedURL(publicBaseURL, secret, fileID string, ttl time.Duration, now time.Time) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("media: public_base_url is required for signed URLs")
	}
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("media: sign secret is required for signed URLs")
	}
	if strings.TrimSpace(fileID) == "" {
		return "", fmt.Errorf("media: file id is required")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	exp := now.Add(ttl).Unix()
	sig := SignFile(secret, fileID, exp)
	u, err := url.Parse(base + SignedPathPrefix + url.PathEscape(fileID))
	if err != nil {
		return "", fmt.Errorf("media: build signed url: %w", err)
	}
	q := u.Query()
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// SignFile returns hex HMAC-SHA256 for fileID and expiry unix timestamp.
func SignFile(secret, fileID string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fileID))
	_, _ = mac.Write([]byte("|"))
	_, _ = mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignedRequest validates exp and sig query parameters for a file id.
func VerifySignedRequest(secret, fileID, expRaw, sig string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("media: sign secret is not configured")
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil {
		return fmt.Errorf("media: invalid exp")
	}
	if now.Unix() > exp {
		return fmt.Errorf("media: signed url expired")
	}
	want := SignFile(secret, fileID, exp)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		// Also accept URL-safe base64 mistakes from clients by rejecting clearly.
		_ = base64.RawURLEncoding
		return fmt.Errorf("media: invalid signature")
	}
	return nil
}
