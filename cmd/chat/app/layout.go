package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// inputBoxExtraHeight is the vertical chrome added by the rounded input border.
const inputBoxExtraHeight = 2

// syncLayout sizes the scrollable transcript for the current terminal height.
func (m *Model) syncLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	headerH := lipgloss.Height(m.renderTopSection())
	footerH := m.footerHeight()
	transcriptH := max(m.height-headerH-footerH, 3)
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(transcriptH)
	m.input.SetWidth(max(20, m.width-10))
}

func (m *Model) footerHeight() int {
	h := 2 // separator + status bar
	if m.status.PlanMode {
		h++
	}
	if m.hint != "" {
		h++
	}
	if m.phase == PhaseConfirming {
		h += m.confirmFooterHeight()
	}
	if m.phase == PhaseSessionPick {
		h += m.sessionPickerFooterHeight()
	}
	h += m.slashSuggestHeight()
	h += 1 + inputBoxExtraHeight // input row + border chrome
	return h
}

// renderTopSection returns banner/splash content above the transcript.
func (m *Model) renderTopSection() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder
	if m.info != nil {
		writeBuilder(&b, renderCompactHeader(m.width, m.sessionTitle, &m.styles))
		writeBuilder(&b, "\n")
	}

	if m.splashVisible && m.info != nil {
		if strings.TrimSpace(m.sessionTitle) == "" {
			writeBuilder(&b, RenderBanner(m.width, &m.styles))
			writeBuilder(&b, "\n\n")
		}
		host := m.serverHost
		if host == "" && m.client != nil {
			host = m.client.BaseURL()
		}
		writeBuilder(&b, RenderSplash(m.width, m.info, m.sessionID, host, &m.styles))
		writeBuilder(&b, "\n\n")
		if m.welcomeShown {
			writeBuilder(&b, m.styles.Hint.Render("Welcome to Flowbot Agent! Type your message or /help for commands.")+"\n")
			writeBuilder(&b, m.styles.Hint.Render("Tip: Use /file <path> to attach local file content to your next message.")+"\n\n")
		}
	} else if m.info == nil {
		writeBuilder(&b, RenderBanner(m.width, &m.styles))
		writeBuilder(&b, "\n\n")
	}
	return b.String()
}

// renderFooter returns the fixed bottom chrome and input row.
func (m *Model) renderFooter() string {
	var b strings.Builder
	writeBuilder(&b, FormatSeparator(m.width, &m.styles))
	writeBuilder(&b, "\n")
	writeBuilder(&b, RenderStatusBar(m.status, &m.styles))
	writeBuilder(&b, "\n")
	if m.status.PlanMode {
		writeBuilder(&b, m.styles.Warning.Render("● Plan mode — read-only (research & plan). /plan to exit."))
		writeBuilder(&b, "\n")
	}

	if m.hint != "" {
		writeBuilder(&b, m.styles.Hint.Render(m.hint))
		writeBuilder(&b, "\n")
	}

	if m.phase == PhaseConfirming {
		writeBuilder(&b, m.renderConfirmPrompt())
		writeBuilder(&b, "\n")
	}

	if m.phase == PhaseSessionPick {
		writeBuilder(&b, m.renderSessionPicker())
		writeBuilder(&b, "\n")
	}

	if suggest := m.renderSlashSuggestions(); suggest != "" {
		writeBuilder(&b, suggest)
	}

	writeBuilder(&b, m.renderInputRow())
	return b.String()
}

// renderInputRow wraps the prompt and textarea in a bordered input box.
func (m *Model) renderInputRow() string {
	inner := m.styles.InputPrompt.Render(inputPromptText) + m.input.View()
	box := m.styles.InputBox
	switch m.phase {
	case PhaseStreaming:
		box = box.BorderForeground(colorAccent)
	case PhaseConfirming, PhaseSessionPick:
		box = box.BorderForeground(colorBorder)
	default:
		box = box.BorderForeground(colorPrimary)
	}
	return box.Width(m.width).Render(inner)
}

// renderCompactHeader returns a single-line title rule when the transcript is active.
func renderCompactHeader(width int, sessionTitle string, styles *Styles) string {
	left := styles.BannerTitle.Render(compactBanner)
	leftW := lipgloss.Width(left)

	sessionTitle = strings.TrimSpace(sessionTitle)
	right := ""
	rightW := 0
	if sessionTitle != "" {
		right = styles.Hint.Render(sessionTitle)
		rightW = lipgloss.Width(right)
		const minRule = 1
		available := width - leftW - minRule - 2
		if available <= 0 {
			right = ""
			rightW = 0
		} else if rightW > available {
			right = truncateStyledRunes(sessionTitle, available, styles.Hint)
			rightW = lipgloss.Width(right)
		}
	}

	spacing := 1
	if rightW > 0 {
		spacing = 2
	}
	ruleWidth := max(width-leftW-rightW-spacing, 0)
	rule := styles.Rule.Render(strings.Repeat("─", ruleWidth))

	if rightW == 0 {
		return left + " " + rule
	}
	return left + " " + rule + " " + right
}

func truncateStyledRunes(text string, maxWidth int, style lipgloss.Style) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(text)
	for len(runes) > 0 {
		candidate := style.Render(string(runes) + "…")
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}
