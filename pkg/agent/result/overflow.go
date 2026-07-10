package result

import (
	"errors"
	"regexp"
)

var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)prompt is too long`),
	regexp.MustCompile(`(?i)request_too_large`),
	regexp.MustCompile(`(?i)exceeds the context window`),
	regexp.MustCompile(`(?i)exceeds (?:the )?(?:model'?s )?maximum context length of [\d,]+ tokens?`),
	regexp.MustCompile(`(?i)input token count.*exceeds the maximum`),
	regexp.MustCompile(`(?i)maximum prompt length is \d+`),
	regexp.MustCompile(`(?i)reduce the length of the messages`),
	regexp.MustCompile(`(?i)maximum context length is \d+ tokens`),
	regexp.MustCompile(`(?i)exceeds (?:the )?maximum allowed input length of [\d,]+ tokens?`),
	regexp.MustCompile(`(?i)exceeds the available context size`),
	regexp.MustCompile(`(?i)greater than the context length`),
	regexp.MustCompile(`(?i)context window exceeds limit`),
	regexp.MustCompile(`(?i)exceeded model token limit`),
	regexp.MustCompile(`(?i)too large for model with \d+ maximum context length`),
	regexp.MustCompile(`(?i)prompt too long; exceeded (?:max )?context length`),
	regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`),
	regexp.MustCompile(`(?i)too many tokens`),
	regexp.MustCompile(`(?i)token limit exceeded`),
}

var nonOverflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(Throttling error|Service unavailable):`),
	regexp.MustCompile(`(?i)rate limit`),
	regexp.MustCompile(`(?i)too many requests`),
}

// IsContextOverflowErr reports whether an LLM error indicates context window overflow.
func IsContextOverflowErr(err error) bool {
	if err == nil {
		return false
	}
	var overflow OverflowError
	if errors.As(err, &overflow) {
		return true
	}
	errText := err.Error()
	if matchesOverflowPattern(nonOverflowPatterns, errText) {
		return false
	}
	return matchesOverflowPattern(overflowPatterns, errText)
}

// WrapOverflowError wraps an underlying error as a typed OverflowError when overflow is detected.
func WrapOverflowError(err error) error {
	if err == nil || !IsContextOverflowErr(err) {
		return err
	}
	return NewOverflowError("context window exceeded", err)
}

// MatchesOverflowText reports whether text looks like a context overflow message.
func MatchesOverflowText(text string) bool {
	if IsNonOverflowText(text) {
		return false
	}
	return matchesOverflowPattern(overflowPatterns, text)
}

// IsNonOverflowText reports whether text matches known non-overflow provider errors.
func IsNonOverflowText(text string) bool {
	return matchesOverflowPattern(nonOverflowPatterns, text)
}

func matchesOverflowPattern(patterns []*regexp.Regexp, text string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}
