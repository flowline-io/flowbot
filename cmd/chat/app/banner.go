// Package app implements the flowbot-chat terminal UI.
package app

import (
	"strings"
)

const compactBanner = "Flowbot Agent"

// RenderBanner returns the top ASCII banner, compact when width is narrow.
func RenderBanner(width int, styles Styles) string {
	if width < 60 {
		return styles.BannerTitle.Render(compactBanner)
	}
	lines := []string{
		"  _____ _                    ____        _   ",
		" |  ___| | _____      _____ | __ )  ___ | |_ ",
		" | |_  | |/ _ \\ \\ /\\ / / _ \\|  _ \\ / _ \\| __|",
		" |  _| | | (_) \\ V  V / (_) | |_) | (_) | |_ ",
		" |_|   |_|\\___/ \\_/\\_/ \\___/|____/ \\___/ \\__|",
	}
	var b strings.Builder
	for _, line := range lines {
		writeBuilder(&b, styles.BannerTitle.Render(line))
		_ = b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
