package ctxmgr

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/tmc/langchaingo/llms"
)

// StatefulAgent is the seam ctxmgr needs to read and rewrite in-flight agent state
// without importing the loop/runtime package.
type StatefulAgent interface {
	State() *msg.Context
	ApplyState(fn func(*msg.Context))
}

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
	Tools         []llms.Tool
	ThinkingLevel string
}

// Manager orchestrates compaction, branch summarization, and context budget checks.
type Manager struct {
	model         llms.Model
	modelName     string
	contextWindow int
	settings      Settings
	systemPrompt  string
	tools         []llms.Tool
	thinkingLevel string
}

// New creates a context manager for harness integration.
func New(opts Options) *Manager {
	return &Manager{
		model:         opts.Model,
		modelName:     opts.ModelName,
		contextWindow: opts.ContextWindow,
		settings:      opts.Settings.WithDefaults(),
		systemPrompt:  opts.SystemPrompt,
		tools:         append([]llms.Tool(nil), opts.Tools...),
		thinkingLevel: opts.ThinkingLevel,
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

// UpdateTools replaces the tool schemas forwarded on summarization requests.
func (m *Manager) UpdateTools(tools []llms.Tool) {
	m.tools = append([]llms.Tool(nil), tools...)
}

// UpdateThinkingLevel replaces the thinking level forwarded on summarization requests.
func (m *Manager) UpdateThinkingLevel(level string) {
	m.thinkingLevel = level
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
func (m *Manager) EnsureWithinBudget(ctx context.Context, sess *session.Session, ag StatefulAgent) error {
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
	_, err = m.compactPath(ctx, sess, ag, path, CompactOpts{Force: false}, usage.Tokens)
	return err
}

// CompactAndReload compacts the current branch and reloads agent state.
func (m *Manager) CompactAndReload(ctx context.Context, sess *session.Session, ag StatefulAgent, opts CompactOpts) (CompactReport, error) {
	if sess == nil {
		return CompactReport{}, fmt.Errorf("ctxmgr: nil session")
	}
	if !opts.Force && !m.settings.Enabled {
		return CompactReport{}, nil
	}
	path, err := sess.GetBranch(ctx, "")
	if err != nil {
		return CompactReport{}, fmt.Errorf("ctxmgr: load branch: %w", err)
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
	ag StatefulAgent,
	path []session.TreeEntry,
	opts CompactOpts,
	contextTokens int,
) (CompactReport, error) {
	outcome, err := m.applyPrune(ctx, sess, ag, path, contextTokens)
	if err != nil {
		return outcome.report, err
	}
	if !opts.Force && outcome.report.Pruned && !ShouldCompact(outcome.contextTokens, m.contextWindow, m.settings) {
		return outcome.report, m.reloadKeepingExtras(ctx, sess, ag, outcome.extras)
	}
	return m.summarizeAndPersist(ctx, sess, ag, outcome.path, outcome.extras, opts, outcome.contextTokens, outcome.report)
}

type pruneOutcome struct {
	extras        []msg.AgentMessage
	report        CompactReport
	path          []session.TreeEntry
	contextTokens int
}

func (m *Manager) applyPrune(
	ctx context.Context,
	sess *session.Session,
	ag StatefulAgent,
	path []session.TreeEntry,
	contextTokens int,
) (pruneOutcome, error) {
	out := pruneOutcome{path: path, contextTokens: contextTokens}
	rawExtra := agentExtraMessages(ag, path)
	out.extras = PruneToolOutputs(rawExtra, m.settings)
	if EstimateContextTokens(rawExtra).Tokens != EstimateContextTokens(out.extras).Tokens {
		out.report.Pruned = true
	}

	pruned, err := persistPrunedToolResults(ctx, sess, path, m.settings)
	if err != nil {
		return out, err
	}
	out.report.Pruned = pruned || out.report.Pruned
	if pruned {
		out.path, err = sess.GetBranch(ctx, "")
		if err != nil {
			return out, fmt.Errorf("ctxmgr: load pruned branch: %w", err)
		}
	}
	if out.report.Pruned {
		out.contextTokens = m.GetContextUsage(out.path).Tokens + EstimateContextTokens(out.extras).Tokens
	}
	return out, nil
}

func (m *Manager) summarizeAndPersist(
	ctx context.Context,
	sess *session.Session,
	ag StatefulAgent,
	path []session.TreeEntry,
	extra []msg.AgentMessage,
	opts CompactOpts,
	contextTokens int,
	report CompactReport,
) (CompactReport, error) {
	preparationResult := PrepareCompaction(path, m.settings, PrepareOptions{
		Force:         opts.Force,
		ExtraMessages: extra,
	})
	if !preparationResult.IsOk() {
		_, adaptErr := result.GetOrError(preparationResult)
		return report, adaptErr
	}
	preparation := preparationResult.Value()
	if preparation == nil {
		if report.Pruned {
			return report, m.reloadKeepingExtras(ctx, sess, ag, extra)
		}
		if ShouldCompact(contextTokens, m.contextWindow, m.settings) || opts.Force {
			return report, ErrCompactionRequired
		}
		return report, nil
	}
	preparation.SystemPrompt = m.systemPrompt
	preparation.Tools = m.tools
	preparation.ThinkingLevel = m.thinkingLevel
	compactResult := RunCompaction(ctx, m.model, m.modelName, preparation)
	if !compactResult.IsOk() {
		_, adaptErr := result.GetOrError(compactResult)
		if report.Pruned {
			if reloadErr := m.reloadKeepingExtras(ctx, sess, ag, extra); reloadErr != nil {
				return report, reloadErr
			}
		}
		return report, adaptErr
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
		return report, fmt.Errorf("ctxmgr: persist compaction: %w", err)
	}
	report.Summarized = true
	if ag != nil {
		return report, m.ReloadAgentState(ctx, sess, ag)
	}
	return report, nil
}

func (m *Manager) reloadKeepingExtras(
	ctx context.Context,
	sess *session.Session,
	ag StatefulAgent,
	extras []msg.AgentMessage,
) error {
	if ag == nil {
		return nil
	}
	if err := m.ReloadAgentState(ctx, sess, ag); err != nil {
		return err
	}
	if len(extras) == 0 {
		return nil
	}
	ag.ApplyState(func(state *msg.Context) {
		state.Messages = append(state.Messages, extras...)
	})
	return nil
}

func agentExtraMessages(ag StatefulAgent, path []session.TreeEntry) []msg.AgentMessage {
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

func (m *Manager) ReloadAgentState(ctx context.Context, sess *session.Session, ag StatefulAgent) error {
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
