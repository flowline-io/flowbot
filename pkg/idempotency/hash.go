package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/bytedance/sonic"
)

var errUnsupportedNumber = errors.New("canonical JSON number must be finite")

// Hash returns the SHA-256 hex digest of v encoded as RFC 8785 canonical JSON.
// A nil payload hashes as {}.
func Hash(v any) (string, error) {
	if v == nil {
		v = map[string]any{}
	}
	raw, err := sonic.Marshal(v)
	if err != nil {
		return "", err
	}
	return HashJSON(raw)
}

// HashJSON hashes a JSON payload. A blank payload hashes as {}.
// JSON null hashes as null. Object key order does not change the digest. Array order does.
func HashJSON(raw []byte) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}")
	}
	var parsed any
	if err := sonic.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	canon, err := appendCanonical(nil, parsed)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}

func appendCanonical(dst []byte, v any) ([]byte, error) {
	switch n := v.(type) {
	case nil:
		return append(dst, "null"...), nil
	case bool:
		if n {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil
	case string:
		raw, err := sonic.Marshal(n)
		if err != nil {
			return nil, err
		}
		return append(dst, raw...), nil
	case float64:
		next, err := appendNumber(dst, n)
		if err != nil {
			return nil, err
		}
		return next, nil
	case []any:
		return appendCanonicalArray(dst, n)
	case map[string]any:
		return appendCanonicalObject(dst, n)
	default:
		raw, err := sonic.Marshal(n)
		if err != nil {
			return nil, err
		}
		var parsed any
		if err := sonic.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		return appendCanonical(dst, parsed)
	}
}

func appendCanonicalArray(dst []byte, items []any) ([]byte, error) {
	dst = append(dst, '[')
	for i, item := range items {
		if i > 0 {
			dst = append(dst, ',')
		}
		var err error
		dst, err = appendCanonical(dst, item)
		if err != nil {
			return nil, err
		}
	}
	return append(dst, ']'), nil
}

func appendCanonicalObject(dst []byte, obj map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, compareUTF16)
	dst = append(dst, '{')
	for i, k := range keys {
		if i > 0 {
			dst = append(dst, ',')
		}
		raw, err := sonic.Marshal(k)
		if err != nil {
			return nil, err
		}
		dst = append(dst, raw...)
		dst = append(dst, ':')
		dst, err = appendCanonical(dst, obj[k])
		if err != nil {
			return nil, err
		}
	}
	return append(dst, '}'), nil
}

func appendNumber(dst []byte, f float64) ([]byte, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, errUnsupportedNumber
	}
	if f == 0 {
		return append(dst, '0'), nil
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		return append(dst, es6Exponent(f)...), nil
	}
	return strconv.AppendFloat(dst, f, 'f', -1, 64), nil
}

// es6Exponent is the ECMAScript NumberToString exponent form used by RFC 8785.
// The exponent has a sign and no leading zeros.
func es6Exponent(f float64) string {
	s := strconv.FormatFloat(f, 'e', -1, 64)
	i := strings.LastIndexByte(s, 'e')
	if i < 0 || i+2 >= len(s) {
		return s
	}
	sign := s[i+1]
	digits := strings.TrimLeft(s[i+2:], "0")
	if digits == "" {
		digits = "0"
	}
	return s[:i+1] + string(sign) + digits
}

func compareUTF16(a, b string) int {
	aa := utf16.Encode([]rune(a))
	bb := utf16.Encode([]rune(b))
	n := min(len(bb), len(aa))
	for i := range n {
		if aa[i] < bb[i] {
			return -1
		}
		if aa[i] > bb[i] {
			return 1
		}
	}
	return len(aa) - len(bb)
}
