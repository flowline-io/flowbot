package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
)

const (
	letters  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	urlRegex = `https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
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

func Masker(input string, start int) string {
	if len(input) <= start {
		return input
	}
	lenStart := len(input[start:])
	switch {
	case lenStart <= 3:
		return input[:start] + strings.Repeat("*", lenStart)
	case 3 < lenStart && lenStart <= 5:
		return input[:start+1] + strings.Repeat("*", lenStart-2) + input[lenStart+start-1:]
	case 5 < lenStart && lenStart <= 10:
		return input[:start+2] + strings.Repeat("*", lenStart-4) + input[lenStart+start-2:]
	case lenStart > 10:
		return input[:start+4] + strings.Repeat("*", lenStart-8) + input[lenStart+start-4:]
	default:
		return ""
	}
}

func Fn(public any) string {
	switch v := public.(type) {
	case map[string]any:
		if s, ok := v["fn"].(string); ok {
			return s
		}
	}
	return ""
}

func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func MD5(txt string) string {
	h := md5.New()
	_, _ = h.Write(StringToBytes(txt))
	return hex.EncodeToString(h.Sum(nil))
}

func SHA1(txt string) string {
	h := sha1.New()
	_, _ = h.Write(StringToBytes(txt))
	return hex.EncodeToString(h.Sum(nil))
}

func MarkdownTitle(txt string) string {
	lines := strings.Split(txt, "\n")
	if len(lines) > 0 {
		first := strings.TrimLeft(lines[0], "#")
		first = strings.TrimSpace(first)
		return first
	}
	return ""
}

func NewUUID() string {
	u := uuid.New()
	return u.String()
}

func ValidImageContentType(ct string) bool {
	return strings.HasPrefix(ct, "image/")
}

func FileAndLine() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s:%d", file, line)
}

func PrettyPrintJsonStyle(data any) {
	d, err := sonic.MarshalIndent(data, "", "  ")
	if err != nil {
		_, _ = fmt.Printf("error: %s, data: %+v\n", err, data)
		return
	}
	_, _ = fmt.Println(string(d))
}

func PrettyPrintYamlStyle(data any) {
	d, err := yaml.Marshal(data)
	if err != nil {
		_, _ = fmt.Printf("error: %s, data: %+v\n", err, data)
		return
	}
	_, _ = fmt.Println(string(d))
}

func YamlToJson(data []byte) ([]byte, error) {
	var def map[string]any
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}

	jsonData, err := sonic.Marshal(def)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
