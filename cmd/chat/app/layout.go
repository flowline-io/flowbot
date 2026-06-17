package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

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
	m.input.SetWidth(max(20, m.width-6))
}

func (m *Model) footerHeight() int {
	h := 3 // separators + status bar
	if m.hint != "" {
		h++
	}
	if m.phase == PhaseConfirming {
		h += m.confirmFooterHeight()
	}
	h += m.slashSuggestHeight()
	h++ // input row
	return h
}

// renderTopSection returns banner/splash content above the transcript.
func (m *Model) renderTopSection() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder
	if m.splashVisible && m.info != nil {
		writeBuilder(&b, RenderBanner(m.width, &m.styles))
		writeBuilder(&b, "\n\n")
		host := m.serverHost
		if host == "" && m.client != nil {
			host = m.client.BaseURL()
		}
		writeBuilder(&b, RenderSplash(m.width, m.info, m.sessionID, host, &m.styles))
		writeBuilder(&b, "\n\n")
		if m.welcomeShown {
			writeBuilder(&b, "Welcome to Flowbot Agent! Type your message or /help for commands.\n")
			writeBuilder(&b, "Tip: Use /file <path> to attach local file content to your next message.\n\n")
		}
	} else if m.transcript.Len() > 0 {
		writeBuilder(&b, m.styles.BannerTitle.Render(compactBanner))
		writeBuilder(&b, "\n\n")
	} else {
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
	writeBuilder(&b, FormatSeparator(m.width, &m.styles))
	writeBuilder(&b, "\n")

	if m.hint != "" {
		writeBuilder(&b, m.styles.Hint.Render(m.hint))
		writeBuilder(&b, "\n")
	}

	if m.phase == PhaseConfirming {
		writeBuilder(&b, m.renderConfirmPrompt())
		writeBuilder(&b, "\n")
	}

	if suggest := m.renderSlashSuggestions(); suggest != "" {
		writeBuilder(&b, suggest)
	}

	writeBuilder(&b, m.styles.InputPrompt.Render(inputPromptText))
	writeBuilder(&b, m.input.View())
	return b.String()
}
