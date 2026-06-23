package app

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/flowline-io/flowbot/pkg/client"
)

type streamDoneMsg struct{ err error }

type tickMsg time.Time

type initDoneMsg struct {
	info         *client.ChatAgentInfo
	sessionID    string
	sessionTitle string
	mode         string
	err          string
}

// sessionModeMsg reports the result of a /plan toggle.
type sessionModeMsg struct {
	mode string
	err  string
}

// sessionModeLoadMsg reports the mode fetched after switching sessions.
type sessionModeLoadMsg struct {
	sessionID    string
	mode         string
	sessionTitle string
	err          string
}

// sessionNewMsg reports the result of an async /new command.
type sessionNewMsg struct {
	id  string
	err string
}

// sessionEndMsg reports the result of an async /end command.
type sessionEndMsg struct {
	err  string
	warn string
}

// sessionCompactMsg reports the result of an async /compact command.
type sessionCompactMsg struct {
	compacted    bool
	tokensBefore int
	tokensAfter  int
	err          string
}

const chatRequestTimeout = 30 * time.Second

// chatCompactionTimeout allows LLM summarization to finish; longer than routine API calls.
const chatCompactionTimeout = 3 * time.Minute

// Model is the bubbletea model for flowbot-chat.
type Model struct {
	client  *client.Client
	profile string
	styles  Styles

	width  int
	height int

	info         *client.ChatAgentInfo
	sessionID    string
	sessionTitle string
	mode         string
	serverHost   string

	viewport viewport.Model
	input    textarea.Model

	transcript    strings.Builder
	splashVisible bool
	welcomeShown  bool

	phase   RunPhase
	stream  streamRunState
	confirm confirmUIState
	picker  sessionPickerUIState

	pendingFile *FileAttachment
	hint        string
	toolPreview string

	slashMatches []SlashCommand
	slashPick    int

	inputHist inputHistory

	status StatusSnapshot

	startedAt    time.Time
	turnStarted  time.Time
	messageCount int

	errMsg   string
	quitting bool

	selActive   bool
	selDragging bool
	selAnchor   textPos
	selFocus    textPos
}

// NewModel constructs the chat TUI model.
func NewModel(cl *client.Client, profile string) *Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.Placeholder = "Type your message or /help for commands..."
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.SetVirtualCursor(true)
	styles := textarea.DefaultDarkStyles()
	styles.Cursor.Blink = true
	ta.SetStyles(styles)
	ta.Focus()

	vp := viewport.New()
	vp.SetContent("")

	return &Model{
		client:        cl,
		profile:       profile,
		styles:        NewStyles(),
		input:         ta,
		viewport:      vp,
		splashVisible: true,
		welcomeShown:  true,
		phase:         PhaseIdle,
		startedAt:     time.Now(),
		hint:          defaultHint(),
		inputHist:     inputHistory{index: -1},
	}
}

const sessionModePlan = "plan"

func defaultHint() string {
	return inputHintFor("")
}

// inputHintFor returns the input-area hint for the given session mode.
func inputHintFor(mode string) string {
	if mode == sessionModePlan {
		return "Plan mode (read-only) · /plan to exit · /help"
	}
	return "/help · /new · /file · Ctrl+C quit"
}

// applySessionModeDisplay syncs plan-mode indicators across status bar and hint line.
func (m *Model) applySessionModeDisplay(mode string) {
	if mode == "" {
		mode = "normal"
	}
	m.mode = mode
	m.status.PlanMode = mode == sessionModePlan
	m.hint = inputHintFor(mode)
}

func (m *Model) resetInputHint() {
	m.hint = inputHintFor(m.mode)
}

// finalizeSessionMode applies server mode to UI chrome and respects session cache hints.
func (m *Model) finalizeSessionMode(mode string) {
	m.applySessionModeDisplay(mode)
	if hint := sessionCacheHint(SaveSessionID(m.profile, m.sessionID)); hint != "" {
		m.hint = hint
	}
	m.syncLayout()
	m.syncViewport()
}

// Init loads agent info and resumes any saved session.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.initCmd(), tickCmd(), m.focusInputCmd())
}

func (m *Model) initCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		info, err := m.client.ChatAgent.Info(ctx)
		if err != nil {
			return initDoneMsg{err: err.Error()}
		}
		sessionID, err := resumeSessionID(ctx, m.client, m.profile)
		if err != nil {
			return initDoneMsg{err: err.Error()}
		}
		var mode string
		var sessionTitle string
		if sessionID != "" {
			sessionInfo, err := m.client.ChatAgent.GetSessionMode(ctx, sessionID)
			if err != nil {
				return initDoneMsg{err: err.Error()}
			}
			mode = sessionInfo.Mode
			sessionTitle = sessionInfo.Title
		}
		return initDoneMsg{info: info, sessionID: sessionID, sessionTitle: sessionTitle, mode: mode}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func renderTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// NewProgram wraps the model in a bubbletea program.
func NewProgram(m *Model) *tea.Program {
	return tea.NewProgram(m)
}
