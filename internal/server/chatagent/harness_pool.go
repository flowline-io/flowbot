package chatagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/ctxmgr"
	"github.com/flowline-io/flowbot/pkg/agent/harness"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

type pooledHarness struct {
	harness    *harness.Harness
	configHash string
	promptVer  uint64
	lastUsed   atomic.Int64
}

func (e *pooledHarness) touchLastUsed() {
	e.lastUsed.Store(time.Now().UnixNano())
}

func (e *pooledHarness) staleAt(now time.Time, ttl time.Duration) bool {
	last := e.lastUsed.Load()
	if last == 0 {
		return true
	}
	return now.Sub(time.Unix(0, last)) > ttl
}

var harnessPool sync.Map

// EvictHarnessPool removes a cached harness for the given session.
func EvictHarnessPool(sessionID string) {
	harnessPool.Delete(sessionID)
}

// AbortSessionHarness cooperatively cancels the agent loop for a pooled session harness.
func AbortSessionHarness(sessionID string) {
	raw, ok := harnessPool.Load(sessionID)
	if !ok {
		return
	}
	entry, ok := raw.(*pooledHarness)
	if !ok || entry.harness == nil {
		return
	}
	entry.harness.Agent().Abort()
}

// ResetHarnessPoolForTest clears all pooled harnesses.
func ResetHarnessPoolForTest() {
	harnessPool = sync.Map{}
}

func getOrCreateHarness(ctx context.Context, req RunRequest, textLen int) (*harness.Harness, error) {
	evictStaleHarnesses()

	var h *harness.Harness
	if raw, ok := harnessPool.Load(req.SessionID); ok {
		entry, ok := raw.(*pooledHarness)
		if !ok {
			harnessPool.Delete(req.SessionID)
		} else {
			entry.touchLastUsed()
			if refreshed, err := refreshPooledHarness(ctx, req, entry, textLen); err != nil {
				return nil, err
			} else if refreshed != nil {
				harnessPool.Store(req.SessionID, refreshed)
				h = refreshed.harness
			} else {
				harnessPool.Store(req.SessionID, entry)
				h = entry.harness
			}
		}
	}

	if h == nil {
		built, err := buildRunHarness(ctx, req, textLen)
		if err != nil {
			return nil, err
		}
		created := &pooledHarness{
			harness:    built.harness,
			configHash: built.configHash,
			promptVer:  built.promptVer,
		}
		created.touchLastUsed()
		harnessPool.Store(req.SessionID, created)
		h = built.harness
	}

	if err := applySessionMode(ctx, h, req); err != nil {
		return nil, err
	}
	return h, nil
}

func applySessionMode(ctx context.Context, h *harness.Harness, req RunRequest) error {
	mode := LoadSessionMode(ctx, req.SessionID)
	kind := req.Kind
	if kind == "" {
		kind = RunKindInteractive
	}

	baseTools := BaseToolNamesForRun(kind, req.Tools)
	scopedTools := ApplyToolScope(ToolScopeInput{
		Mode:      mode,
		Kind:      kind,
		UserText:  req.Text,
		AllActive: baseTools,
	})
	h.SetActiveTools(activeSubagentTools(scopedTools, req.Skills))

	workspace, err := WorkspaceFromConfig()
	if err != nil {
		return err
	}
	systemPrompt := SessionSystemPrompt(ctx, workspace, mode)
	if len(req.Skills) > 0 {
		systemPrompt = buildFilteredSystemPrompt(ctx, workspace, req.Skills)
	}
	h.SetSystemPrompt(systemPrompt)
	if ctxMgr := h.ContextManager(); ctxMgr != nil {
		ctxMgr.UpdateSystemPrompt(systemPrompt)
	}
	return nil
}

type builtHarness struct {
	harness    *harness.Harness
	configHash string
	promptVer  uint64
}

func refreshPooledHarness(ctx context.Context, req RunRequest, entry *pooledHarness, textLen int) (*pooledHarness, error) {
	workspace, err := WorkspaceFromConfig()
	if err != nil {
		return nil, err
	}
	currentHash, err := harnessConfigHash(workspace)
	if err != nil {
		return nil, err
	}
	if currentHash != entry.configHash {
		EvictHarnessPool(req.SessionID)
		built, err := buildRunHarness(ctx, req, textLen)
		if err != nil {
			return nil, err
		}
		refreshed := &pooledHarness{
			harness:    built.harness,
			configHash: built.configHash,
			promptVer:  built.promptVer,
		}
		refreshed.touchLastUsed()
		return refreshed, nil
	}

	currentPromptVer := PromptCacheVersion()
	if currentPromptVer != entry.promptVer {
		systemPrompt := SystemPrompt(ctx, workspace)
		entry.harness.SetSystemPrompt(systemPrompt)
		if ctxMgr := entry.harness.ContextManager(); ctxMgr != nil {
			ctxMgr.UpdateSystemPrompt(systemPrompt)
		}
		entry.promptVer = currentPromptVer
	}
	return nil, nil
}

func evictStaleHarnesses() {
	now := time.Now()
	harnessPool.Range(func(key, value any) bool {
		entry, ok := value.(*pooledHarness)
		if !ok {
			harnessPool.Delete(key)
			return true
		}
		if entry.staleAt(now, sessionLockTTL) {
			harnessPool.Delete(key)
		}
		return true
	})
}

func harnessConfigHash(workspace coding.Workspace) (string, error) {
	cfg, chatModel, toolModel, dual, err := agentLoopConfig()
	if err != nil {
		return "", err
	}
	compaction := config.App.ChatAgent.Compaction
	sandbox := config.App.ChatAgent.Sandbox
	abilityParts := make([]string, 0, len(config.App.ChatAgent.AbilityTools))
	for _, entry := range config.App.ChatAgent.AbilityTools {
		abilityParts = append(abilityParts, fmt.Sprintf("%s:%s:%s:%t",
			strings.TrimSpace(entry.Name),
			strings.TrimSpace(entry.Capability),
			strings.TrimSpace(entry.Operation),
			entry.Readonly,
		))
	}
	parts := []string{
		workspace.Root,
		chatModel,
		toolModel,
		fmt.Sprintf("dual=%t", dual),
		fmt.Sprintf("max_steps=%d", cfg.MaxSteps),
		fmt.Sprintf("compaction=%t:%t:%d:%d",
			compaction.AutoEnabled(),
			compaction.PruneEnabled(),
			compaction.ReservedTokens(),
			compaction.KeepRecentBudget(),
		),
		fmt.Sprintf("sandbox=%t:%s:%s:%s",
			sandbox.Enabled,
			strings.TrimSpace(sandbox.Image),
			strings.TrimSpace(sandbox.Network),
			strings.TrimSpace(sandbox.Memory),
		),
		"ability=" + strings.Join(abilityParts, ","),
		promptConfigHash(workspace.Root),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:]), nil
}

func buildRunHarness(ctx context.Context, req RunRequest, textLen int) (*builtHarness, error) {
	workspace, err := WorkspaceFromConfig()
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] workspace config session=%s: %w", req.SessionID, err))
		return nil, err
	}

	cfg, chatModel, toolModel, dual, err := agentLoopConfig()
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] model config session=%s: %w", req.SessionID, err))
		return nil, fmt.Errorf("chat agent models: %w", err)
	}

	llmModel, resolvedName, err := NewModelForTest(ctx, chatModel)
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] model init session=%s model=%s: %w", req.SessionID, chatModel, err))
		return nil, fmt.Errorf("chat agent model: %w", err)
	}

	uid, uidErr := SessionOwnerUID(ctx, req.SessionID)
	if uidErr != nil {
		uid = types.Uid("")
	}
	kind := req.Kind
	if kind == "" {
		kind = RunKindInteractive
	}
	registry, err := NewRegistry(workspace, &TaskToolDeps{
		SessionID: req.SessionID,
		UID:       uid,
		Kind:      kind,
	}, &ScheduleToolDeps{SessionID: req.SessionID, UID: uid})
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] tool registry session=%s: %w", req.SessionID, err))
		return nil, err
	}

	agentSession := session.New(NewDBStorage(req.SessionID, uid, TokenUsageSourceFromRunKind(kind)))
	branch, err := agentSession.GetBranch(ctx, "")
	if err != nil {
		flog.Error(fmt.Errorf("[chat-agent] load branch session=%s: %w", req.SessionID, err))
		return nil, fmt.Errorf("load session branch: %w", err)
	}

	systemPrompt := SystemPrompt(ctx, workspace)
	agentCtx := session.ToAgentContext(session.BuildContext(branch), systemPrompt)
	contextWindow := config.ChatAgentContextWindow()
	compactionSettings := ctxmgr.SettingsFromConfig(config.App.ChatAgent.Compaction)
	ctxManager := ctxmgr.New(ctxmgr.Options{
		Model:         llmModel,
		ModelName:     resolvedName,
		ContextWindow: contextWindow,
		Settings:      compactionSettings,
		SystemPrompt:  systemPrompt,
	})

	flog.Debug("[chat-agent] harness prompt session=%s model=%s dual_model=%t chat_model=%s tool_model=%s workspace=%s branch_entries=%d max_steps=%d text_len=%d context_window=%d compaction_enabled=%t",
		req.SessionID, resolvedName, dual, chatModel, toolModel, workspace.Root, len(branch), cfg.MaxSteps, textLen, contextWindow, compactionSettings.Enabled)

	hookRegistry := hooks.NewRegistry()
	RegisterHooks(hookRegistry, ChatHookDeps{
		SessionID:   req.SessionID,
		UID:         uid,
		SessionMode: LoadSessionMode(ctx, req.SessionID),
		Kind:        kind,
	})

	configHash, err := harnessConfigHash(workspace)
	if err != nil {
		return nil, err
	}

	return &builtHarness{
		harness: harness.New(harness.Options{
			AgentOptions: agent.Options{
				InitialState: agentCtx,
				Config:       cfg,
				Model:        llmModel,
				Registry:     registry,
			},
			Session:        agentSession,
			SystemPrompt:   systemPrompt,
			ModelName:      chatModel,
			ContextManager: ctxManager,
			Hooks:          hookRegistry,
		}),
		configHash: configHash,
		promptVer:  PromptCacheVersion(),
	}, nil
}
