package app

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/client"
)

const contextBarWidth = 20

const (
	blockFull    = '⛁'
	blockPartial = '⛀'
	blockEmpty   = '⛶'
	blockBuffer  = '⛝'
)

// RenderContextUsage formats the /context panel from a server usage report.
func RenderContextUsage(info *client.ChatContextUsage, styles Styles) string {
	if info == nil {
		return styles.Hint.Render("Context usage unavailable")
	}

	var b strings.Builder
	writeBuilder(&b, " Context Usage\n")

	usageBar := renderContextUsageBar(info.TotalPercent, contextBarWidth)
	modelLabel := fmt.Sprintf("%s[%s]", info.Model, formatContextWindow(info.ContextWindow))
	writeBuilder(&b, fmt.Sprintf("     %s   %s\n", usageBar, modelLabel))

	summary := fmt.Sprintf("%s/%s tokens (%s)",
		formatContextTokenCount(info.TotalTokens),
		formatContextWindow(info.ContextWindow),
		formatContextPercent(info.TotalPercent),
	)
	writeBuilder(&b, fmt.Sprintf("     %s   %s\n", emptyContextBar(contextBarWidth), summary))
	writeBuilder(&b, fmt.Sprintf("     %s   Estimated usage by category\n", emptyContextBar(contextBarWidth)))

	for _, cat := range info.Categories {
		if cat.ID == "autocompact_buffer" {
			bar := renderAutocompactBar(cat.Percent, contextBarWidth)
			writeBuilder(&b, fmt.Sprintf("     %s   %c %s: %s tokens (%s)\n",
				bar, blockBuffer, cat.Label,
				formatContextTokenCount(cat.Tokens),
				formatContextPercent(cat.Percent),
			))
			continue
		}
		icon := blockFull
		if cat.ID == "free_space" {
			icon = blockEmpty
		}
		writeBuilder(&b, fmt.Sprintf("     %s   %c %s: %s tokens (%s)\n",
			emptyContextBar(contextBarWidth), icon, cat.Label,
			formatContextTokenCount(cat.Tokens),
			formatContextPercent(cat.Percent),
		))
	}

	if len(info.Skills) > 0 {
		writeBuilder(&b, "\n     Skills · /skills\n\n")
		writeBuilder(&b, "     Built-in\n")
		for i, skill := range info.Skills {
			prefix := "├"
			if i == len(info.Skills)-1 {
				prefix = "└"
			}
			writeBuilder(&b, fmt.Sprintf("     %s %s: %s\n", prefix, skill.Name, formatSkillTokenEstimate(skill.Tokens)))
		}
	}

	return styles.Hint.Render(b.String())
}

func renderContextUsageBar(percent float64, width int) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	totalUnits := percent / 100 * float64(width)
	full := int(totalUnits)
	hasPartial := totalUnits-float64(full) >= 0.15
	if full >= width {
		return joinContextBlocks(width, 0, 0)
	}
	if hasPartial && full < width {
		return joinContextBlocks(full, 1, width-full-1)
	}
	return joinContextBlocks(full, 0, width-full)
}

func renderAutocompactBar(percent float64, width int) string {
	if width <= 0 {
		return ""
	}
	bufferBlocks := int(percent/100*float64(width) + 0.5)
	if bufferBlocks <= 0 && percent > 0 {
		bufferBlocks = 1
	}
	if bufferBlocks > width {
		bufferBlocks = width
	}
	return renderBufferTail(width-bufferBlocks, bufferBlocks)
}

func renderBufferTail(empty, buffer int) string {
	parts := make([]string, 0, empty+buffer)
	for range empty {
		parts = append(parts, string(blockEmpty))
	}
	for range buffer {
		parts = append(parts, string(blockBuffer))
	}
	return strings.Join(parts, " ")
}

func joinContextBlocks(full, partial, empty int) string {
	parts := make([]string, 0, full+partial+empty)
	for range full {
		parts = append(parts, string(blockFull))
	}
	for range partial {
		parts = append(parts, string(blockPartial))
	}
	for range empty {
		parts = append(parts, string(blockEmpty))
	}
	return strings.Join(parts, " ")
}

func emptyContextBar(width int) string {
	return joinContextBlocks(0, 0, width)
}

func formatContextTokenCount(tokens int) string {
	switch {
	case tokens >= 1_000_000:
		val := float64(tokens) / 1_000_000
		if val >= 10 {
			return fmt.Sprintf("%.0fm", val)
		}
		return fmt.Sprintf("%.1fm", val)
	case tokens >= 1000:
		val := float64(tokens) / 1000
		if val >= 100 {
			return fmt.Sprintf("%.0fk", val)
		}
		return fmt.Sprintf("%.1fk", val)
	default:
		return fmt.Sprintf("%d", tokens)
	}
}

func formatSkillTokenEstimate(tokens int) string {
	if tokens < 20 {
		return "< 20 tokens"
	}
	return fmt.Sprintf("~%s tokens", formatContextTokenCount(tokens))
}

func formatContextWindow(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%dm", n/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
