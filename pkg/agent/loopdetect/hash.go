package loopdetect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

const maxResultHashBytes = 4096

var (
	volatileMetaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpid[=:\s]+\d+`),
		regexp.MustCompile(`(?i)\bduration[_a-z]*[=:\s]+\d+(\.\d+)?m?s?`),
		regexp.MustCompile(`(?i)\belapsed[=:\s]+\d+(\.\d+)?m?s?`),
		regexp.MustCompile(`(?i)\bsession[_-]?id[=:\s]+\S+`),
	}
	exitCodePattern = regexp.MustCompile(`(?i)\bexit[_-]?code[=:\s]+(-?\d+)`)
)

// ArgsHash returns a stable fingerprint for tool name + args.
func ArgsHash(toolName string, args map[string]any) string {
	h := sha256.New()
	_, _ = h.Write([]byte(toolName))
	_, _ = h.Write([]byte{0})
	if len(args) == 0 {
		return hex.EncodeToString(h.Sum(nil))
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
		raw, err := sonic.Marshal(args[k])
		if err != nil {
			_, _ = h.Write(fmt.Append(nil, args[k]))
		} else {
			_, _ = h.Write(raw)
		}
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ResultHash returns a stable fingerprint for a completed tool result.
func ResultHash(toolName string, result msg.ToolResultMessage) string {
	body := resultText(result)
	body = strings.TrimSpace(body)
	if len(body) > maxResultHashBytes {
		body = body[:maxResultHashBytes]
	}
	switch toolName {
	case "run_terminal", "run_code":
		body = stabilizeShellResult(body)
	}
	h := sha256.New()
	if result.IsError {
		_, _ = h.Write([]byte("err:1"))
	} else {
		_, _ = h.Write([]byte("err:0"))
	}
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func resultText(result msg.ToolResultMessage) string {
	var b strings.Builder
	for _, part := range result.Parts {
		switch p := part.(type) {
		case msg.TextPart:
			_, _ = b.WriteString(p.Text)
		default:
			raw, err := sonic.Marshal(part)
			if err == nil {
				_, _ = b.Write(raw)
			}
		}
	}
	return b.String()
}

func stabilizeShellResult(body string) string {
	exit := ""
	if m := exitCodePattern.FindStringSubmatch(body); len(m) >= 2 {
		exit = m[1]
	}
	cleaned := body
	for _, re := range volatileMetaPatterns {
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if exit != "" {
		return "exit=" + exit + " " + cleaned
	}
	return cleaned
}
