package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
)

// runControlMsg reports the result of an async cancel or confirm API call.
type runControlMsg struct {
	err       error
	okHint    string
	failLabel string
}

// runControlHint builds the footer status line after a cancel/confirm API call.
func runControlHint(err error, okHint, failLabel string) string {
	if err != nil {
		return fmt.Sprintf("%s: %v", failLabel, err)
	}
	return okHint
}

func (m *Model) cancelRunCmd() tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	parent := m.streamRequestCtx()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, chatRequestTimeout)
		defer cancel()
		err := cl.ChatAgent.Cancel(ctx, sessionID)
		return runControlMsg{
			err:       err,
			okHint:    "Canceled — ready for input",
			failLabel: "Cancel failed — server may still be running",
		}
	}
}

func (m *Model) denyConfirmCmd(confirmID string) tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	parent := m.streamRequestCtx()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, chatRequestTimeout)
		defer cancel()
		err := cl.ChatAgent.ConfirmWithMode(ctx, sessionID, confirmID, false, client.ConfirmModeReject, "")
		return runControlMsg{
			err:       err,
			okHint:    "Canceled — ready for input",
			failLabel: "Confirm failed — server may still be waiting",
		}
	}
}

func (m *Model) submitConfirmChoiceCmd(id, pattern string, choice confirmChoice) tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	parent := m.streamRequestCtx()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, chatRequestTimeout)
		defer cancel()
		err := cl.ChatAgent.ConfirmWithMode(ctx, sessionID, id, choice.approved, choice.mode, pattern)
		return runControlMsg{
			err:       err,
			okHint:    "Ctrl+C cancel run",
			failLabel: "Confirm failed — server may still be waiting",
		}
	}
}

func (m *Model) updateRunControl(msg runControlMsg) (*Model, tea.Cmd) {
	m.hint = runControlHint(msg.err, msg.okHint, msg.failLabel)
	return m, m.focusInputCmd()
}
