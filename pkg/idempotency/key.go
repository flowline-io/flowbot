// Package idempotency validates write idempotency keys and hashes business payloads.
package idempotency

import (
	"strings"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/pkg/types"
)

const maxKeyBytes = 255

// Validate reports whether key is a client idempotency key.
// Empty means the caller did not send one. A non-empty key must be 1 to 255
// bytes of ASCII letters, digits, or . _ ~ -.
func Validate(key string) error {
	if key == "" {
		return nil
	}
	if len(key) > maxKeyBytes || !utf8.ValidString(key) {
		return invalidKey()
	}
	for i := 0; i < len(key); i++ {
		if !allowedKeyByte(key[i]) {
			return invalidKey()
		}
	}
	return nil
}

// Take removes idempotency_key from args and validates it.
// A missing or blank value returns an empty key. The remaining args are the business payload.
func Take(args map[string]any) (string, error) {
	if args == nil {
		return "", nil
	}
	raw, ok := args["idempotency_key"]
	if !ok || raw == nil {
		return "", nil
	}
	delete(args, "idempotency_key")
	text, ok := raw.(string)
	if !ok {
		return "", invalidKey()
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	if err := Validate(text); err != nil {
		return "", err
	}
	return text, nil
}

func invalidKey() error {
	return &types.Error{
		Kind:    types.ErrInvalidArgument,
		Message: "idempotency key must be 1 to 255 bytes of ASCII letters, digits, or . _ ~ -",
		Field:   "idempotency_key",
	}
}

func allowedKeyByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '.' || b == '_' || b == '~' || b == '-':
		return true
	default:
		return false
	}
}
