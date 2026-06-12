package app

import (
	"charm.land/lipgloss/v2"
)

var (
	colorBannerTitle = lipgloss.Color("#FFBF00")
	colorBannerDim   = lipgloss.Color("#B8860B")
	colorBorder      = lipgloss.Color("#FFD700")
	colorMuted       = lipgloss.Color("#888888")
	colorUser        = lipgloss.Color("#87CEEB")
	colorAssistant   = lipgloss.Color("#E6E6FA")
	colorWarning     = lipgloss.Color("#FFA500")
	colorConfirm     = lipgloss.Color("#FFD700")
	colorStatusOK    = lipgloss.Color("#50FA7B")
	colorStatusWarn  = lipgloss.Color("#F1FA8C")
	colorStatusCrit  = lipgloss.Color("#FF5555")
	colorStatusText  = lipgloss.Color("#CCCCCC")
	colorInputPrompt = lipgloss.Color("#98FB98")
)

// Styles holds lipgloss styles for the chat UI.
type Styles struct {
	BannerTitle lipgloss.Style
	BannerDim   lipgloss.Style
	SplashBox   lipgloss.Style
	Rule        lipgloss.Style
	UserMsg     lipgloss.Style
	Assistant   lipgloss.Style
	Hint        lipgloss.Style
	Warning     lipgloss.Style
	ConfirmBox  lipgloss.Style
	Status      lipgloss.Style
	InputPrompt lipgloss.Style
}

// NewStyles builds the default dark-theme styles.
func NewStyles() Styles {
	return Styles{
		BannerTitle: lipgloss.NewStyle().Foreground(colorBannerTitle).Bold(true),
		BannerDim:   lipgloss.NewStyle().Foreground(colorBannerDim),
		SplashBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),
		Rule:        lipgloss.NewStyle().Foreground(colorMuted),
		UserMsg:     lipgloss.NewStyle().Foreground(colorUser).Bold(true),
		Assistant:   lipgloss.NewStyle().Foreground(colorAssistant),
		Hint:        lipgloss.NewStyle().Foreground(colorMuted).Italic(true),
		Warning:     lipgloss.NewStyle().Foreground(colorWarning).Bold(true),
		ConfirmBox:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorConfirm).Padding(0, 1),
		Status:      lipgloss.NewStyle().Foreground(colorStatusText),
		InputPrompt: lipgloss.NewStyle().Foreground(colorInputPrompt).Bold(true),
	}
}
