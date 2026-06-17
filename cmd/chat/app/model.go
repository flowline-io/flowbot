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

// streamMsg delivers one SSE event into the bubbletea event loop.
type streamMsg client.ChatStreamEvent

type streamDoneMsg struct{ err error }

type tickMsg time.Time

type initDoneMsg struct {
	info      *client.ChatAgentInfo
	sessionID string
	err       string
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

const chatRequestTimeout = 30 * time.Second

// Model is the bubbletea model for flowbot-chat.
type Model struct {
	client  *client.Client
	profile string
	styles  Styles

	width  int
	height int

	info       *client.ChatAgentInfo
	sessionID  string
	serverHost string

	viewport viewport.Model
	input    textarea.Model

	transcript    strings.Builder
	streamOverlay strings.Builder
	splashVisible bool
	welcomeShown  bool

	phase            RunPhase
	streamCancel     context.CancelFunc
	streamCtx        context.Context
	rawAssistant     string
	streamingBaseLen int
	renderPending    bool
	renderDeadline   time.Time

	streamCh chan tea.Msg

	pendingConfirmID string
	confirmTool      string
	confirmSummary   string
	confirmPick      int

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

func defaultHint() string {
	return "/help · /new · /file · Ctrl+C quit"
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
		return initDoneMsg{info: info, sessionID: sessionID}
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
