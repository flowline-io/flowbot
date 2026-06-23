package app

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
)

// streamRunState holds in-flight SSE delivery state for one user turn.
type streamRunState struct {
	cancel           context.CancelFunc
	ctx              context.Context
	ch               chan tea.Msg
	overlay          strings.Builder
	rawThinking      string
	rawAssistant     string
	streamingBaseLen int
	renderPending    bool
	renderDeadline   time.Time
}

// confirmUIState holds pending tool-approval UI state.
type confirmUIState struct {
	id               string
	tool             string
	summary          string
	permission       string
	pattern          string
	suggestedPattern string
	suggestAlways    bool
	pick             int
}

// sessionPickerUIState holds the /sessions picker list.
type sessionPickerUIState struct {
	list []client.ChatSessionSummary
	pick int
}

// clearConfirm resets confirmation UI state on the model.
func (m *Model) clearConfirm() {
	m.confirm.clear()
}

func (c *confirmUIState) clear() {
	c.id, c.tool, c.summary = ClearConfirmState()
	c.permission = ""
	c.pattern = ""
	c.suggestedPattern = ""
	c.suggestAlways = false
	c.pick = 0
}

// clearSessionPick resets the session picker UI state.
func (m *Model) clearSessionPick() {
	m.picker.clear()
}

func (p *sessionPickerUIState) clear() {
	p.list = nil
	p.pick = 0
}

// streamRequestCtx returns the active stream context or background when idle.
func (m *Model) streamRequestCtx() context.Context {
	if m.stream.ctx != nil {
		return m.stream.ctx
	}
	return context.Background()
}
