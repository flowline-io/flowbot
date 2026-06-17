package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	agentmsg "github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/client"
)

// Update handles bubbletea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.updateWindowSize(msg)
	case initDoneMsg:
		return m.updateInitDone(msg)
	case hydrateMsg:
		return m.updateHydrate(msg)
	case sessionNewMsg:
		return m.updateSessionNew(msg)
	case sessionEndMsg:
		return m.updateSessionEnd(msg)
	case contextUsageMsg:
		return m.updateContextUsage(msg)
	case sessionCompactMsg:
		return m.updateSessionCompact(msg)
	case sessionExportMsg:
		return m.updateSessionExport(msg)
	case tickMsg:
		return m.updateTick(msg)
	case streamEventMsg:
		return m.updateStreamEvent(msg)
	case streamDoneMsg:
		return m.updateStreamDone(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseWheelMsg, tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg:
		return m.handleMouse(msg)
	default:
		return m.updateDefault(msg)
	}
}

func (m *Model) updateWindowSize(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.syncLayout()
	return m, nil
}

func (m *Model) updateInitDone(msg initDoneMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.errMsg = msg.err
		return m, nil
	}
	if msg.info != nil {
		m.info = msg.info
		m.status.Model = msg.info.ChatModel
		m.status.ContextWindow = ResolveContextWindowFromInfo(msg.info)
	}
	if msg.sessionID != "" {
		m.sessionID = msg.sessionID
		if hint := sessionCacheHint(SaveSessionID(m.profile, msg.sessionID)); hint != "" {
			m.hint = hint
		}
	}
	m.serverHost = m.client.BaseURL()
	if m.sessionID != "" {
		return m, tea.Batch(m.hydrateHistoryCmd(), m.focusInputCmd())
	}
	return m, m.focusInputCmd()
}

func (m *Model) updateHydrate(msg hydrateMsg) (*Model, tea.Cmd) {
	if msg.content != "" {
		writeBuilder(&m.transcript, msg.content)
		m.splashVisible = false
	}
	m.messageCount = msg.count
	m.applyHistoryUsage(msg.estimatedTokens)
	m.syncViewport()
	return m, nil
}

func (m *Model) updateSessionNew(msg sessionNewMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
		return m, m.focusInputCmd()
	}
	m.sessionID = msg.id
	if hint := sessionCacheHint(SaveSessionID(m.profile, msg.id)); hint != "" {
		m.hint = hint
	}
	m.transcript.Reset()
	m.messageCount = 0
	m.resetSessionUsage()
	m.splashVisible = true
	m.syncViewport()
	return m, m.focusInputCmd()
}

func (m *Model) updateSessionEnd(msg sessionEndMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
	} else {
		m.sessionID = ""
		if msg.warn != "" {
			m.hint = msg.warn
		} else {
			m.hint = "Session ended"
		}
	}
	return m, m.focusInputCmd()
}

func (m *Model) updateTick(msg tickMsg) (*Model, tea.Cmd) {
	m.status.Elapsed = time.Since(m.startedAt)
	if m.phase == PhaseStreaming {
		m.status.TurnElapsed = time.Since(m.turnStarted)
		m.status.Streaming = true
		m.status.SpinnerFrame++
	} else {
		m.status.Streaming = false
	}
	if m.renderPending && time.Now().After(m.renderDeadline) {
		m.refreshStreamingAssistant()
		m.renderPending = false
	}
	inputCmd := m.updateInput(msg)
	return m, tea.Batch(tickCmd(), inputCmd)
}

func (m *Model) updateStreamEvent(msg streamEventMsg) (*Model, tea.Cmd) {
	next, cmd := m.handleStreamEvent(msg.event)
	return next, tea.Batch(cmd, next.pumpStream())
}

func (m *Model) updateStreamDone(msg streamDoneMsg) (*Model, tea.Cmd) {
	m.phase = PhaseIdle
	m.status.Streaming = false
	m.hint = defaultHint()
	if msg.err != nil && !m.quitting {
		m.appendSystem(fmt.Sprintf("Error: %v", msg.err))
	}
	if m.rawAssistant != "" {
		m.finalizeAssistant()
	}
	m.streamCh = nil
	m.streamCancel = nil
	m.syncViewport()
	return m, m.focusInputCmd()
}

func (m *Model) updateDefault(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	inputCmd := m.updateInput(msg)
	return m, tea.Batch(cmd, inputCmd)
}

type streamEventMsg struct {
	event client.ChatStreamEvent
}

func (m *Model) handleKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	if m.phase == PhaseConfirming {
		if handled, cmd := m.handleConfirmKey(msg); handled {
			return m, cmd
		}
	}

	switch msg.String() {
	case "ctrl+c":
		_, action := HandleCtrlC(m.phase)
		switch action {
		case CtrlCQuit:
			m.quitting = true
			if m.streamCancel != nil {
				m.streamCancel()
			}
			return m, tea.Quit
		case CtrlCCancelRun:
			if m.streamCancel != nil {
				m.streamCancel()
			}
			m.phase = PhaseIdle
			m.hint = runControlHint(
				m.client.ChatAgent.Cancel(context.Background(), m.sessionID),
				"Canceled — ready for input",
				"Cancel failed — server may still be running",
			)
			return m, nil
		case CtrlCDenyConfirm:
			id := m.pendingConfirmID
			m.clearConfirm()
			m.phase = PhaseIdle
			m.hint = runControlHint(
				m.client.ChatAgent.Confirm(context.Background(), m.sessionID, id, false),
				"Canceled — ready for input",
				"Confirm failed — server may still be waiting",
			)
			return m, nil
		}
	case "enter":
		if m.phase == PhaseStreaming {
			return m, nil
		}
		if m.slashSuggestActive() {
			text, run := m.acceptSlashSelection()
			if !run {
				return m, nil
			}
			m.inputHist.push(text)
			m.input.Reset()
			m.clearSlashSuggest()
			return m.handleUserInput(text)
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		m.inputHist.push(text)
		m.input.Reset()
		m.clearSlashSuggest()
		return m.handleUserInput(text)
	}

	return m, m.updateInput(msg)
}

func (m *Model) handleUserInput(text string) (*Model, tea.Cmd) {
	if cmd, args, ok := ParseSlashCommand(text); ok {
		return m.handleSlash(cmd, args)
	}
	m.appendUser(text)
	payload := WrapUserMessage(m.pendingFile, text)
	m.pendingFile = nil
	m.hint = defaultHint()
	return m.startStream(payload)
}

func (m *Model) handleSlash(cmd, args string) (*Model, tea.Cmd) {
	switch cmd {
	case "help":
		m.appendSystem(SlashHelp())
		return m, m.focusInputCmd()
	case "quit":
		m.quitting = true
		if m.streamCancel != nil {
			m.streamCancel()
		}
		return m, tea.Quit
	case "clear":
		m.transcript.Reset()
		m.splashVisible = true
		m.syncViewport()
		return m, nil
	case "file":
		return m.handleSlashFile(args)
	case "new":
		return m, m.sessionNewCmd()
	case "end":
		return m, m.sessionEndCmd()
	case "status":
		m.appendSystem(SessionStatusText(m.sessionID, m.messageCount))
		return m, m.focusInputCmd()
	case "context":
		return m, m.contextUsageCmd()
	case "compact":
		return m, m.sessionCompactCmd()
	case "resume":
		m.transcript.Reset()
		m.streamOverlay.Reset()
		m.messageCount = 0
		return m, m.hydrateHistoryCmd()
	case "export":
		return m, m.sessionExportCmd(args)
	case "auth":
		return m.handleSlashAuth(args)
	default:
		m.hint = "Unknown command; try /help"
		return m, nil
	}
}

func (m *Model) handleSlashFile(args string) (*Model, tea.Cmd) {
	att, err := ReadLocalFile(args)
	if err != nil {
		m.hint = err.Error()
		return m, nil
	}
	m.pendingFile = &att
	if w := FormatFileWarning(att); w != "" {
		m.hint = w
	} else {
		m.hint = fmt.Sprintf("Attached %s", att.Path)
	}
	return m, nil
}

func (m *Model) handleSlashAuth(args string) (*Model, tea.Cmd) {
	if args == "status" || args == "" {
		m.appendSystem(fmt.Sprintf("profile=%s server=%s", m.profile, m.client.BaseURL()))
	} else {
		m.appendSystem("Use 'flowbot-cli login' to configure auth")
	}
	return m, m.focusInputCmd()
}

func (m *Model) startStream(text string) (*Model, tea.Cmd) {
	if m.sessionID == "" {
		m.hint = "No active session"
		return m, nil
	}
	m.phase = PhaseStreaming
	m.turnStarted = time.Now()
	m.rawAssistant = ""
	m.streamingBaseLen = m.transcript.Len()
	m.streamOverlay.Reset()
	m.hint = "Ctrl+C cancel run"
	m.splashVisible = false
	m.appendSystem("Initializing agent...")
	m.syncViewport()

	ctx, cancel := context.WithCancel(context.Background())
	m.streamCtx = ctx
	m.streamCancel = cancel
	m.streamCh = make(chan tea.Msg, 64)

	cl := m.client
	sessionID := m.sessionID
	ch := m.streamCh
	go func() {
		err := cl.ChatAgent.SendMessageSSE(ctx, sessionID, text, func(ev client.ChatStreamEvent) error {
			msg := streamEventMsg{event: ev}
			if ev.Type != "delta" {
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case ch <- msg:
						return nil
					}
				}
			}
			// Delta events are best-effort: drop when the pump channel is full.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- msg:
			default:
			}
			return nil
		})
		done := streamDoneMsg{err: err}
		for {
			select {
			case <-ctx.Done():
				return
			case ch <- done:
				return
			}
		}
	}()

	return m, m.pumpStream()
}

func (m *Model) pumpStream() tea.Cmd {
	if m.streamCh == nil {
		return nil
	}
	ch := m.streamCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return msg
	}
}

func (m *Model) handleStreamEvent(ev client.ChatStreamEvent) (*Model, tea.Cmd) {
	switch ev.Type {
	case "delta":
		m.rawAssistant = agentmsg.SanitizeAssistantDisplayText(ev.Text)
		m.renderPending = true
		m.renderDeadline = time.Now().Add(RenderDebounce(m.rawAssistant))
	case "tool":
		line := fmt.Sprintf("Running tool: %s...", ev.Name)
		if ev.Stdout != "" {
			m.toolPreview = ev.Name + ": " + ev.Stdout
			line = m.toolPreview
		}
		m.appendSystem(line)
	case "usage":
		m.status.TotalTokens = ev.TotalTokens
		if ev.ContextWindow > 0 {
			m.status.ContextWindow = ev.ContextWindow
		} else if m.status.ContextWindow <= 0 {
			m.status.ContextWindow = m.effectiveContextWindow()
		}
		m.status.ContextPercent = contextUsagePercent(ev.TotalTokens, m.status.ContextWindow, ev.ContextPercent)
	case "confirm":
		m.phase = PhaseConfirming
		m.pendingConfirmID = ev.ID
		m.confirmTool = ev.Tool
		m.confirmSummary = ev.Summary
		m.confirmPick = confirmChoiceApprove
		m.hint = ""
	case "confirm_resolved":
		m.clearConfirm()
		if m.phase == PhaseConfirming {
			m.phase = PhaseStreaming
			m.hint = "Ctrl+C cancel run"
		}
	case "canceled":
		m.phase = PhaseIdle
		m.hint = "Canceled — ready for input"
		m.clearConfirm()
	case "done":
		if ev.Text != "" {
			m.rawAssistant = agentmsg.SanitizeAssistantDisplayText(ev.Text)
		}
		m.finalizeAssistant()
		m.phase = PhaseIdle
		m.hint = defaultHint()
	case "error":
		m.appendSystem("Error: " + ev.Message)
		m.phase = PhaseIdle
		m.hint = defaultHint()
	}
	m.syncViewport()
	var cmd tea.Cmd
	if m.renderPending {
		cmd = renderTickCmd(RenderDebounce(m.rawAssistant))
	}
	return m, cmd
}

func (m *Model) clearConfirm() {
	m.pendingConfirmID, m.confirmTool, m.confirmSummary = ClearConfirmState()
	m.confirmPick = confirmChoiceApprove
}

func (m *Model) appendUser(text string) {
	if m.transcript.Len() > 0 {
		writeBuilder(&m.transcript, FormatSeparator(m.width, &m.styles)+"\n")
	}
	writeBuilder(&m.transcript, FormatHistoryLine("user", text, &m.styles))
}

func (m *Model) appendSystem(text string) {
	line := FormatSystemLine(text, &m.styles)
	if m.phase == PhaseStreaming || m.phase == PhaseConfirming {
		writeBuilder(&m.streamOverlay, line)
		return
	}
	writeBuilder(&m.transcript, line)
	m.splashVisible = false
	m.syncViewport()
}

func (m *Model) refreshStreamingAssistant() {
	if m.rawAssistant == "" {
		return
	}
	transcript := m.transcript.String()
	baseLen := min(m.streamingBaseLen,
		// Transcript may shrink mid-stream (e.g. /clear); clamp to avoid slice panic.
		len(transcript))
	base := transcript[:baseLen]
	rendered := FormatAssistantBlock(m.rawAssistant, m.width-2, &m.styles)
	m.transcript.Reset()
	writeBuilder(&m.transcript, base)
	writeBuilder(&m.transcript, rendered)
	writeBuilder(&m.transcript, m.streamOverlay.String())
}

func (m *Model) finalizeAssistant() {
	m.refreshStreamingAssistant()
	m.rawAssistant = ""
	m.streamOverlay.Reset()
	m.messageCount++
}

func (m *Model) syncViewport() {
	m.syncLayout()
	m.viewport.SetContent(m.transcript.String())
	m.viewport.GotoBottom()
}

type hydrateMsg struct {
	content         string
	count           int
	estimatedTokens int
}

type contextUsageMsg struct {
	usage *client.ChatContextUsage
	err   string
}

func (m *Model) contextUsageCmd() tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	return func() tea.Msg {
		if sessionID == "" {
			return contextUsageMsg{err: "No active session"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		usage, err := cl.ChatAgent.ContextUsage(ctx, sessionID)
		if err != nil {
			return contextUsageMsg{err: err.Error()}
		}
		return contextUsageMsg{usage: usage}
	}
}

func (m *Model) sessionCompactCmd() tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	m.hint = "Compacting..."
	return func() tea.Msg {
		if sessionID == "" {
			return sessionCompactMsg{err: "No active session"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), chatCompactionTimeout)
		defer cancel()
		result, err := cl.ChatAgent.Compact(ctx, sessionID)
		if err != nil {
			return sessionCompactMsg{err: err.Error()}
		}
		return sessionCompactMsg{
			compacted:    result.Compacted,
			tokensBefore: result.TokensBefore,
			tokensAfter:  result.TokensAfter,
		}
	}
}

type sessionExportMsg struct {
	path  string
	count int
	err   string
}

func (m *Model) sessionExportCmd(args string) tea.Cmd {
	sessionID := m.sessionID
	cl := m.client
	return func() tea.Msg {
		if sessionID == "" {
			return sessionExportMsg{err: "No active session"}
		}
		path, err := ResolveExportPath(args, sessionID)
		if err != nil {
			return sessionExportMsg{err: err.Error()}
		}
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		export, err := cl.ChatAgent.ExportSession(ctx, sessionID)
		if err != nil {
			return sessionExportMsg{err: err.Error()}
		}
		if err := WriteSessionExport(path, export); err != nil {
			return sessionExportMsg{err: err.Error()}
		}
		return sessionExportMsg{path: path, count: export.EntryCount}
	}
}

func (m *Model) updateSessionExport(msg sessionExportMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
		return m, m.focusInputCmd()
	}
	m.appendSystem(FormatExportSuccess(msg.path, msg.count))
	return m, m.focusInputCmd()
}

func (m *Model) updateContextUsage(msg contextUsageMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
		return m, m.focusInputCmd()
	}
	m.appendSystem(RenderContextUsage(msg.usage, &m.styles))
	if msg.usage != nil {
		m.status.TotalTokens = msg.usage.TotalTokens
		if msg.usage.ContextWindow > 0 {
			m.status.ContextWindow = msg.usage.ContextWindow
		} else {
			m.status.ContextWindow = m.effectiveContextWindow()
		}
		m.status.ContextPercent = msg.usage.TotalPercent
		if m.status.Model == "" && msg.usage.Model != "" {
			m.status.Model = msg.usage.Model
		}
	}
	return m, m.focusInputCmd()
}

func (m *Model) updateSessionCompact(msg sessionCompactMsg) (*Model, tea.Cmd) {
	if msg.err != "" {
		m.hint = msg.err
		return m, m.focusInputCmd()
	}
	if !msg.compacted {
		m.appendSystem("No older history could be compacted")
		return m, m.focusInputCmd()
	}
	m.hint = formatCompactionSuccess(msg.tokensBefore, msg.tokensAfter)
	m.transcript.Reset()
	m.streamOverlay.Reset()
	m.messageCount = 0
	return m, tea.Batch(m.hydrateHistoryCmd(), m.focusInputCmd())
}

func (m *Model) hydrateHistoryCmd() tea.Cmd {
	sessionID := m.sessionID
	width := m.width
	styles := m.styles
	cl := m.client
	return func() tea.Msg {
		if sessionID == "" {
			return hydrateMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		msgs, err := cl.ChatAgent.ListMessages(ctx, sessionID)
		if err != nil {
			return initDoneMsg{err: err.Error()}
		}
		return hydrateMsg{
			content:         FormatHistoryMessages(msgs, width, &styles),
			count:           len(msgs),
			estimatedTokens: EstimateHistoryTokens(msgs),
		}
	}
}

func (m *Model) sessionNewCmd() tea.Cmd {
	cl := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		id, err := cl.ChatAgent.CreateSession(ctx)
		if err != nil {
			return sessionNewMsg{err: err.Error()}
		}
		return sessionNewMsg{id: id}
	}
}

func (m *Model) sessionEndCmd() tea.Cmd {
	cl := m.client
	sessionID := m.sessionID
	profile := m.profile
	return func() tea.Msg {
		if sessionID == "" {
			return sessionEndMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), chatRequestTimeout)
		defer cancel()
		if err := cl.ChatAgent.CloseSession(ctx, sessionID); err != nil {
			return sessionEndMsg{err: err.Error()}
		}
		if hint := sessionCacheHint(ClearSessionID(profile)); hint != "" {
			return sessionEndMsg{warn: hint}
		}
		return sessionEndMsg{}
	}
}
