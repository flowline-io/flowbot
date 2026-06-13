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
		` _____ _               _           _        _                    _   `,
		`|  ___| | _____      _| |__   ___ | |_     / \   __ _  ___ _ __ | |_ `,
		`| |_  | |/ _ \ \ /\ / / '_ \ / _ \| __|   / _ \ / _| |/ _ \ '_ \| __|`,
		`|  _| | | (_) \ V  V /| |_) | (_) | |_   / ___ \ (_| |  __/ | | | |_ `,
		`|_|   |_|\___/ \_/\_/ |_.__/ \___/ \__| /_/   \_\__, |\___|_| |_|\__|`,
		`                                                |___/                `,
	}
	var b strings.Builder
	for _, line := range lines {
		writeBuilder(&b, styles.BannerTitle.Render(line))
		_ = b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
