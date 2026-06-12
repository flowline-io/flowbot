package ctxmgr

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/tmc/langchaingo/llms"
)

// ContextUsage reports estimated context consumption for UI and hooks.
type ContextUsage struct {
	Tokens        int
	ContextWindow int
	Percent       float64
}

// Options configures a context manager instance.
type Options struct {
	Model         llms.Model
	ModelName     string
	ContextWindow int
	Settings      Settings
	SystemPrompt  string
}

// Manager orchestrates compaction, branch summarization, and context budget checks.
type Manager struct {
	model         llms.Model
	modelName     string
	contextWindow int
	settings      Settings
	systemPrompt  string
}

// New creates a context manager for harness integration.
func New(opts Options) *Manager {
	return &Manager{
		model:         opts.Model,
		modelName:     opts.ModelName,
		contextWindow: opts.ContextWindow,
		settings:      opts.Settings.WithDefaults(),
		systemPrompt:  opts.SystemPrompt,
	}
}

// Settings returns the active compaction settings.
func (m *Manager) Settings() Settings {
	return m.settings
}

// UpdateSystemPrompt replaces the system prompt used for context usage estimates.
func (m *Manager) UpdateSystemPrompt(systemPrompt string) {
	m.systemPrompt = systemPrompt
}

// ContextWindow returns the configured model context window size.
func (m *Manager) ContextWindow() int {
	return m.contextWindow
}

// GetContextUsage estimates current branch context consumption including system prompt overhead.
func (m *Manager) GetContextUsage(path []session.TreeEntry) ContextUsage {
	messages := session.BuildContext(path).Messages
	tokens := EstimateContextTokens(messages).Tokens
	if m.systemPrompt != "" {
		tokens += EstimateTokens(msg.UserMessage{Parts: []msg.ContentPart{msg.TextPart{Text: m.systemPrompt}}})
	}
	usage := ContextUsage{Tokens: tokens, ContextWindow: m.contextWindow}
	if m.contextWindow > 0 {
		usage.Percent = float64(tokens) / float64(m.contextWindow) * 100
	}
	return usage
}

// EnsureWithinBudget compacts session history when usage exceeds the threshold.
func (m *Manager) EnsureWithinBudget(ctx context.Context, sess *session.Session, ag *agent.Agent) error {
	if sess == nil || !m.settings.Enabled {
		return nil
	}
	path, err := sess.GetBranch(ctx, "")
	if err != nil {
		return fmt.Errorf("ctxmgr: load branch: %w", err)
	}
	usage := m.GetContextUsage(path)
	if !ShouldCompact(usage.Tokens, m.contextWindow, m.settings) {
		return nil
	}
	return m.compactPath(ctx, sess, ag, path, CompactOpts{Force: false}, usage.Tokens)
}

// CompactAndReload compacts the current branch and reloads agent state.
func (m *Manager) CompactAndReload(ctx context.Context, sess *session.Session, ag *agent.Agent, opts CompactOpts) error {
	if sess == nil {
		return fmt.Errorf("ctxmgr: nil session")
	}
	if !opts.Force && !m.settings.Enabled {
		return nil
	}
	path, err := sess.GetBranch(ctx, "")
	if err != nil {
		return fmt.Errorf("ctxmgr: load branch: %w", err)
	}
	usage := m.GetContextUsage(path)
	if ag != nil {
		usage.Tokens += EstimateContextTokens(agentExtraMessages(ag, path)).Tokens
	}
	return m.compactPath(ctx, sess, ag, path, opts, usage.Tokens)
}

// MoveTo navigates the session tree, auto-summarizing abandoned branches when needed.
func (m *Manager) MoveTo(ctx context.Context, sess *session.Session, targetEntryID, summary string) error {
	if sess == nil {
		return fmt.Errorf("ctxmgr: nil session")
	}
	if summary != "" {
		return sess.MoveTo(ctx, targetEntryID, summary)
	}

	oldLeaf, err := sess.GetBranch(ctx, "")
	if err != nil {
		return fmt.Errorf("ctxmgr: load current branch: %w", err)
	}
	if len(oldLeaf) == 0 {
		return sess.MoveTo(ctx, targetEntryID, "")
	}
	oldLeafID := oldLeaf[len(oldLeaf)-1].ID
	if oldLeafID == targetEntryID {
		return nil
	}

	allEntries, err := sess.ListEntries(ctx)
	if err != nil {
		return err
	}
	collected := CollectBranchEntries(allEntries, oldLeafID, targetEntryID)
	if !collected.IsOk() {
		_, adaptErr := result.GetOrError(collected)
		return adaptErr
	}
	abandoned := collected.Value().Entries
	messages, fileOps, _ := PrepareBranchSummary(abandoned, m.contextWindow, m.settings)
	if len(messages) == 0 {
		return sess.MoveTo(ctx, targetEntryID, "")
	}
	summaryResult := RunBranchSummary(ctx, m.model, m.modelName, messages, fileOps, m.settings)
	if !summaryResult.IsOk() {
		branchErr := summaryResult.ErrorValue()
		if result.IsCode(branchErr, "aborted") {
			return ErrBranchSummaryAborted
		}
		_, adaptErr := result.GetOrError(summaryResult)
		return fmt.Errorf("ctxmgr: branch summary: %w", adaptErr)
	}
	return sess.MoveTo(ctx, targetEntryID, summaryResult.Value().Summary)
}

func (m *Manager) compactPath(
	ctx context.Context,
	sess *session.Session,
	ag *agent.Agent,
	path []session.TreeEntry,
	opts CompactOpts,
	contextTokens int,
) error {
	extra := agentExtraMessages(ag, path)
	preparationResult := PrepareCompaction(path, m.settings, PrepareOptions{
		Force:         opts.Force,
		ExtraMessages: extra,
	})
	if !preparationResult.IsOk() {
		_, adaptErr := result.GetOrError(preparationResult)
		return adaptErr
	}
	preparation := preparationResult.Value()
	if preparation == nil {
		if ShouldCompact(contextTokens, m.contextWindow, m.settings) || opts.Force {
			return ErrCompactionRequired
		}
		return nil
	}
	compactResult := RunCompaction(ctx, m.model, m.modelName, preparation)
	if !compactResult.IsOk() {
		_, adaptErr := result.GetOrError(compactResult)
		return adaptErr
	}
	compacted := compactResult.Value()
	if err := sess.AppendCompaction(ctx, session.CompactionResult{
		EntryID:          NewCompactionEntryID(),
		Summary:          compacted.Summary,
		FirstKeptEntryID: compacted.FirstKeptEntryID,
		TokensBefore:     compacted.TokensBefore,
		ReadFiles:        compacted.ReadFiles,
		ModifiedFiles:    compacted.ModifiedFiles,
	}); err != nil {
		return fmt.Errorf("ctxmgr: persist compaction: %w", err)
	}
	if ag != nil {
		return m.ReloadAgentState(ctx, sess, ag)
	}
	return nil
}

func agentExtraMessages(ag *agent.Agent, path []session.TreeEntry) []msg.AgentMessage {
	if ag == nil {
		return nil
	}
	sessionMsgs := session.BuildContext(path).Messages
	agentMsgs := ag.State().Messages
	if len(agentMsgs) <= len(sessionMsgs) {
		return nil
	}
	return append([]msg.AgentMessage(nil), agentMsgs[len(sessionMsgs):]...)
}

func (m *Manager) ReloadAgentState(ctx context.Context, sess *session.Session, ag *agent.Agent) error {
	branch, err := sess.GetBranch(ctx, "")
	if err != nil {
		return fmt.Errorf("ctxmgr: reload branch: %w", err)
	}
	sessionCtx := session.BuildContext(branch)
	agentCtx := session.ToAgentContext(sessionCtx, m.systemPrompt)
	ag.ApplyState(func(state *msg.Context) {
		state.SystemPrompt = agentCtx.SystemPrompt
		state.Messages = append([]msg.AgentMessage(nil), agentCtx.Messages...)
		state.ModelName = agentCtx.ModelName
	})
	return nil
}
