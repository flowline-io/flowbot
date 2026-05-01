package ability

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

type CursorPayload struct {
	Capability     string    `json:"capability,omitempty"`
	Backend        string    `json:"backend,omitempty"`
	Strategy       string    `json:"strategy,omitempty"`
	ProviderCursor string    `json:"provider_cursor,omitempty"`
	Page           int       `json:"page,omitempty"`
	Offset         int       `json:"offset,omitempty"`
	Limit          int       `json:"limit,omitempty"`
	SortBy         string    `json:"sort_by,omitempty"`
	SortOrder      string    `json:"sort_order,omitempty"`
	FilterHash     string    `json:"filter_hash,omitempty"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
}

func EncodeCursor(secret []byte, payload CursorPayload) (string, error) {
	if len(secret) == 0 {
		return "", types.Errorf(types.ErrInvalidArgument, "cursor secret is required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", types.WrapError(types.ErrInvalidArgument, "marshal cursor payload", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	signature := signCursor(secret, encoded)
	return encoded + "." + signature, nil
}

func DecodeCursor(secret []byte, cursor string, now time.Time) (CursorPayload, error) {
	var payload CursorPayload
	if len(secret) == 0 {
		return payload, types.Errorf(types.ErrInvalidArgument, "cursor secret is required")
	}
	parts := strings.Split(cursor, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return payload, types.Errorf(types.ErrInvalidArgument, "invalid cursor")
	}
	expected := signCursor(secret, parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return payload, types.Errorf(types.ErrInvalidArgument, "invalid cursor signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, types.WrapError(types.ErrInvalidArgument, "decode cursor", err)
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, types.WrapError(types.ErrInvalidArgument, "unmarshal cursor", err)
	}
	if !payload.ExpiresAt.IsZero() && now.After(payload.ExpiresAt) {
		return payload, types.Errorf(types.ErrInvalidArgument, "cursor expired")
	}
	return payload, nil
}

func signCursor(secret []byte, encodedPayload string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}
