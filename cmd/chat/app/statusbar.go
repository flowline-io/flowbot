package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/flowline-io/flowbot/pkg/agent/model"
)

// StatusSnapshot drives the fixed status bar.
type StatusSnapshot struct {
	Model          string
	TotalTokens    int
	ContextWindow  int
	ContextPercent float64
	Elapsed        time.Duration
	TurnElapsed    time.Duration
	Streaming      bool
	SpinnerFrame   int
}

// RenderStatusBar formats the Hermes-style status line.
func RenderStatusBar(snap StatusSnapshot, styles Styles) string {
	icon := "🤖"
	if snap.Streaming {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		icon = frames[snap.SpinnerFrame%len(frames)]
	}

	window := snap.ContextWindow
	if window <= 0 {
		window = model.ContextWindowFor(snap.Model)
	}
	pct := contextUsagePercent(snap.TotalTokens, window, snap.ContextPercent)
	bar := progressBar(pct, 10)
	color := colorStatusOK
	switch {
	case pct >= 85:
		color = colorStatusCrit
	case pct >= 60:
		color = colorStatusWarn
	}

	barStyled := lipgloss.NewStyle().Foreground(color).Render(bar)
	line := fmt.Sprintf(" %s %s │ %d/%s │ %s %s │ %s │ ⏱ %s",
		icon,
		snap.Model,
		snap.TotalTokens,
		formatTokenWindow(window),
		barStyled,
		formatContextPercent(pct),
		formatDuration(snap.Elapsed),
		formatDuration(snap.TurnElapsed),
	)
	return styles.Status.Render(line)
}

// formatContextPercent renders a human-readable usage percentage.
func formatContextPercent(pct float64) string {
	if pct <= 0 {
		return "0%"
	}
	if pct < 10 {
		return fmt.Sprintf("%.1f%%", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// contextUsagePercent derives display percent from token counts, falling back to server value.
func contextUsagePercent(total, window int, reported float64) float64 {
	if window <= 0 {
		window = model.DefaultContextWindow
	}
	if total > 0 && window > 0 {
		return float64(total) / float64(window) * 100
	}
	return reported
}

func progressBar(percent float64, width int) string {
	if width <= 0 {
		width = 10
	}
	filled := int(percent / 100 * float64(width))
	if percent > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func formatTokenWindow(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%dK", n/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}
