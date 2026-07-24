package utils

import "unicode"

// WordCount returns a rough whitespace-separated word count for display metadata.
func WordCount(content string) int {
	count := 0
	inWord := false
	for _, r := range content {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			count++
			inWord = true
		}
	}
	return count
}
