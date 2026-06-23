package app

import (
	"charm.land/lipgloss/v2"
)

var (
	colorPrimary    = lipgloss.Color("#7AA2F7")
	colorAccent     = lipgloss.Color("#BB9AF7")
	colorBorder     = lipgloss.Color("#3B4261")
	colorMuted      = lipgloss.Color("#565F89")
	colorUser       = lipgloss.Color("#7DCFFF")
	colorAssistant  = lipgloss.Color("#C0CAF5")
	colorWarn       = lipgloss.Color("#E0AF68")
	colorOK         = lipgloss.Color("#9ECE6A")
	colorCrit       = lipgloss.Color("#F7768E")
	colorForeground = lipgloss.Color("#C0CAF5")
	colorSystem     = lipgloss.Color("#6B7394")
)

// Styles holds lipgloss styles for the chat UI.
type Styles struct {
	BannerTitle    lipgloss.Style
	BannerDim      lipgloss.Style
	SplashBox      lipgloss.Style
	Rule           lipgloss.Style
	UserMsg        lipgloss.Style
	UserBar        lipgloss.Style
	UserPanel      lipgloss.Style
	Assistant      lipgloss.Style
	AssistantBar   lipgloss.Style
	AssistantPanel lipgloss.Style
	Thinking       lipgloss.Style
	ThinkingPanel  lipgloss.Style
	Hint           lipgloss.Style
	System         lipgloss.Style
	Warning        lipgloss.Style
	ConfirmBox     lipgloss.Style
	Status         lipgloss.Style
	SegLabel       lipgloss.Style
	SegValue       lipgloss.Style
	SegDivider     lipgloss.Style
	InputPrompt    lipgloss.Style
	InputBox       lipgloss.Style
	ToolLine       lipgloss.Style
	ToolSub        lipgloss.Style
	SectionTitle   lipgloss.Style
}

// NewStyles builds the default Tokyo Night-inspired dark theme.
func NewStyles() Styles {
	return Styles{
		BannerTitle: lipgloss.NewStyle().Foreground(colorPrimary).Bold(true),
		BannerDim:   lipgloss.NewStyle().Foreground(colorMuted),
		SplashBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),
		Rule:    lipgloss.NewStyle().Foreground(colorBorder),
		UserMsg: lipgloss.NewStyle().Foreground(colorUser).Bold(true),
		UserBar: lipgloss.NewStyle().Foreground(colorUser).Bold(true),
		UserPanel: lipgloss.NewStyle().PaddingLeft(1),
		Assistant:    lipgloss.NewStyle().Foreground(colorAssistant),
		AssistantBar: lipgloss.NewStyle().Foreground(colorAccent).Bold(true),
		AssistantPanel: lipgloss.NewStyle().PaddingLeft(1),
		Thinking: lipgloss.NewStyle().Foreground(colorMuted).Italic(true),
		ThinkingPanel: lipgloss.NewStyle().PaddingLeft(1),
		Hint:    lipgloss.NewStyle().Foreground(colorMuted).Italic(true),
		System:  lipgloss.NewStyle().Foreground(colorSystem),
		Warning: lipgloss.NewStyle().Foreground(colorWarn).Bold(true),
		ConfirmBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),
		Status: lipgloss.NewStyle().Foreground(colorForeground),
		SegLabel:    lipgloss.NewStyle().Foreground(colorPrimary).Bold(true),
		SegValue:    lipgloss.NewStyle().Foreground(colorMuted),
		SegDivider:  lipgloss.NewStyle().Foreground(colorBorder),
		InputPrompt: lipgloss.NewStyle().Foreground(colorPrimary).Bold(true),
		InputBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),
		ToolLine:     lipgloss.NewStyle().Foreground(colorForeground),
		ToolSub:      lipgloss.NewStyle().Foreground(colorMuted),
		SectionTitle: lipgloss.NewStyle().Foreground(colorAccent).Bold(true),
	}
}
