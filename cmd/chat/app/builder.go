package app

import "strings"

// writeBuilder appends text to a strings.Builder, ignoring write errors from in-memory buffers.
func writeBuilder(b *strings.Builder, s string) {
	_, _ = b.WriteString(s)
}
