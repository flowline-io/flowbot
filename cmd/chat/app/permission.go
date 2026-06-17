package app

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/client"
)

type confirmChoice struct {
	label    string
	mode     client.ConfirmMode
	approved bool
}

func (m *Model) activeConfirmChoices() []confirmChoice {
	choices := []confirmChoice{
		{label: "Approve once", mode: client.ConfirmModeOnce, approved: true},
	}
	if m.confirmSuggestAlways {
		choices = append(choices, confirmChoice{label: "Always allow", mode: client.ConfirmModeAlways, approved: true})
	}
	choices = append(choices, confirmChoice{label: "Deny", mode: client.ConfirmModeReject, approved: false})
	return choices
}

func (m *Model) renderConfirmPrompt() string {
	if m.phase != PhaseConfirming {
		return ""
	}
	var body strings.Builder
	writeBuilder(&body, "Confirm tool "+m.confirmTool)
	writeBuilder(&body, "\n")
	writeBuilder(&body, m.confirmSummary)
	if m.confirmPermission != "" {
		writeBuilder(&body, "\n")
		writeBuilder(&body, fmt.Sprintf("permission: %s", m.confirmPermission))
	}
	if m.confirmPattern != "" {
		writeBuilder(&body, "\n")
		writeBuilder(&body, fmt.Sprintf("pattern: %s", m.confirmPattern))
	}
	if m.confirmSuggestAlways && m.confirmSuggestedPattern != "" {
		writeBuilder(&body, "\n")
		writeBuilder(&body, fmt.Sprintf("always: %s", m.confirmSuggestedPattern))
	}
	writeBuilder(&body, "\n")
	choices := m.activeConfirmChoices()
	for i, choice := range choices {
		line := "  " + choice.label
		if i == m.confirmPick {
			writeBuilder(&body, m.styles.InputPrompt.Render("▸ "+choice.label))
		} else {
			writeBuilder(&body, m.styles.Hint.Render(line))
		}
		writeBuilder(&body, "\n")
	}
	writeBuilder(&body, m.styles.Hint.Render("  ↑↓ select · Enter confirm · Esc deny · Ctrl+C cancel"))
	return m.styles.ConfirmBox.Render(body.String())
}

func (m *Model) confirmFooterHeight() int {
	if m.phase != PhaseConfirming {
		return 0
	}
	lines := 4 + len(m.activeConfirmChoices())
	return lines + 2
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.phase != PhaseConfirming {
		return false, nil
	}
	choices := m.activeConfirmChoices()
	switch msg.Key().Code {
	case tea.KeyUp:
		m.confirmPick = (m.confirmPick + len(choices) - 1) % len(choices)
		return true, nil
	case tea.KeyDown:
		m.confirmPick = (m.confirmPick + 1) % len(choices)
		return true, nil
	case tea.KeyEnter, tea.KeyTab:
		if m.confirmPick < 0 || m.confirmPick >= len(choices) {
			return true, nil
		}
		return true, m.submitConfirmChoice(choices[m.confirmPick])
	case tea.KeyEscape:
		return true, m.submitConfirmChoice(confirmChoice{mode: client.ConfirmModeReject, approved: false})
	}
	switch msg.String() {
	case "y", "Y":
		return true, m.submitConfirmChoice(confirmChoice{mode: client.ConfirmModeOnce, approved: true})
	case "n", "N":
		return true, m.submitConfirmChoice(confirmChoice{mode: client.ConfirmModeReject, approved: false})
	}
	return false, nil
}

func (m *Model) submitConfirmChoice(choice confirmChoice) tea.Cmd {
	id := m.pendingConfirmID
	pattern := ""
	if choice.mode == client.ConfirmModeAlways {
		pattern = m.confirmSuggestedPattern
	}
	m.clearConfirm()
	m.phase = PhaseStreaming
	m.hint = runControlHint(
		m.client.ChatAgent.ConfirmWithMode(context.Background(), m.sessionID, id, choice.approved, choice.mode, pattern),
		"Ctrl+C cancel run",
		"Confirm failed — server may still be waiting",
	)
	return m.focusInputCmd()
}

func confirmChoiceLabel(pick int, choices []confirmChoice) string {
	if pick < 0 || pick >= len(choices) {
		return ""
	}
	return choices[pick].label
}

type permissionMsg struct {
	text string
	err  string
}

func (m *Model) handleSlashPermission(args string) (*Model, tea.Cmd) {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return m, m.fetchPermissions("")
	}
	switch fields[0] {
	case "show":
		key := ""
		if len(fields) > 1 {
			key = fields[1]
		}
		return m, m.fetchPermissions(key)
	case "set":
		raw := strings.TrimSpace(strings.TrimPrefix(args, "set"))
		if raw == "" {
			m.hint = "Usage: /permission set <json>"
			return m, nil
		}
		return m, m.putPermissions(raw)
	case "reset":
		return m, m.resetPermissions()
	case "grants":
		if len(fields) > 1 && fields[1] == "clear" {
			return m, m.clearPermissionGrants()
		}
		return m, m.fetchPermissions("grants")
	default:
		m.hint = "Unknown /permission subcommand; try /permission show"
		return m, nil
	}
}

func (m *Model) fetchPermissions(scope string) tea.Cmd {
	return func() tea.Msg {
		view, err := m.client.ChatAgent.GetPermissions(context.Background(), m.sessionID)
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		text, err := formatPermissionsView(view, scope)
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		return permissionMsg{text: text}
	}
}

func (m *Model) putPermissions(raw string) tea.Cmd {
	return func() tea.Msg {
		var rules map[string]any
		if err := sonic.UnmarshalString(raw, &rules); err != nil {
			return permissionMsg{err: "invalid json: " + err.Error()}
		}
		view, err := m.client.ChatAgent.PutPermissions(context.Background(), rules)
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		text, err := formatPermissionsView(view, "")
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		return permissionMsg{text: "Permissions updated.\n" + text}
	}
}

func (m *Model) resetPermissions() tea.Cmd {
	return func() tea.Msg {
		view, err := m.client.ChatAgent.DeletePermissions(context.Background())
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		text, err := formatPermissionsView(view, "")
		if err != nil {
			return permissionMsg{err: err.Error()}
		}
		return permissionMsg{text: "Permissions reset to defaults.\n" + text}
	}
}

func (m *Model) clearPermissionGrants() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.ChatAgent.ClearPermissionGrants(context.Background(), m.sessionID); err != nil {
			return permissionMsg{err: err.Error()}
		}
		return permissionMsg{text: "Session permission grants cleared."}
	}
}

func formatPermissionsView(view *client.ChatPermissionsView, scope string) (string, error) {
	if view == nil {
		return "", fmt.Errorf("empty permissions view")
	}
	switch scope {
	case "grants":
		data, err := sonic.MarshalString(view.SessionGrants)
		if err != nil {
			return "", err
		}
		return data, nil
	case "":
		data, err := sonic.MarshalString(view.Effective)
		if err != nil {
			return "", err
		}
		return data, nil
	default:
		if src, ok := view.Effective[scope]; ok {
			data, err := sonic.MarshalString(src)
			if err != nil {
				return "", err
			}
			return data, nil
		}
		return "", fmt.Errorf("permission key %q not found", scope)
	}
}

func (m *Model) applyPermissionMsg(msg permissionMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.appendSystem("Permission error: " + msg.err)
	} else {
		m.appendSystem(msg.text)
	}
	m.splashVisible = false
	m.syncViewport()
	return m, m.focusInputCmd()
}
