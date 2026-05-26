package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unsafe"

	"github.com/google/uuid"
)

const (
	letters  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	urlRegex = `https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&/=]*)`
)

func HasHan(txt string) bool {
	for _, runeValue := range txt {
		if unicode.Is(unicode.Han, runeValue) {
			return true
		}
	}
	return false
}

func RandomString(n int) (string, error) {
	ret := make([]byte, n)
	for i := range n {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

func IsUrl(text string) bool {
	re := regexp.MustCompile("^" + urlRegex + "$")
	return re.MatchString(text)
}

func SHA256(txt string) string {
	h := sha256.New()
	_, _ = h.Write(StringToBytes(txt))
	return hex.EncodeToString(h.Sum(nil))
}

func NewUUID() string {
	u := uuid.New()
	return u.String()
}

func ValidImageContentType(ct string) bool {
	return strings.HasPrefix(ct, "image/")
}

// BytesToString converts byte slice to string without allocation.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice without allocation.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
