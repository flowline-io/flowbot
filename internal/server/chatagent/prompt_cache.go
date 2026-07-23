package chatagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const promptCacheTTL = 60 * time.Second

type promptCacheEntry struct {
	prompt            string
	loadedAt          time.Time
	configHash        string
	skillsMaxRev      time.Time
	subagentsMaxRev   time.Time
	memoryFingerprint string
	memoryScope       string
	fileMTimes        map[string]time.Time
}

var (
	promptCacheMu  sync.RWMutex
	promptCache    promptCacheEntry
	promptCacheVer atomic.Uint64
)

// ResetPromptCacheForTest clears the in-process system prompt cache.
func ResetPromptCacheForTest() {
	promptCacheMu.Lock()
	defer promptCacheMu.Unlock()
	promptCache = promptCacheEntry{}
	promptCacheVer.Store(0)
}

// InvalidatePromptCache clears the cached system prompt so the next request rebuilds it.
func InvalidatePromptCache() {
	promptCacheMu.Lock()
	defer promptCacheMu.Unlock()
	promptCache = promptCacheEntry{}
	promptCacheVer.Add(1)
}

// PromptCacheVersion returns a monotonic version token for prompt cache invalidation.
func PromptCacheVersion() uint64 {
	return promptCacheVer.Load()
}

// CachedSystemPrompt returns the chat assistant system prompt, reusing a process cache when inputs are unchanged.
func CachedSystemPrompt(ctx context.Context, ws coding.Workspace) string {
	configHash := promptConfigHash(ws.Root)
	fileMTimes := collectContextFileMTimes(ws.Root, config.App.ChatAgent.ContextFiles)
	skillsMaxRev, err := loadSkillsMaxUpdatedAt(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills revision: %v", err)
	}
	subagentsMaxRev, err := loadSubagentsMaxUpdatedAt(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load subagents revision: %v", err)
	}
	memoryScope := resolveToolMemoryScope(ctx)
	memoryFP := loadMemoryFactsFingerprint(ctx, memoryScope)

	promptCacheMu.RLock()
	cached := promptCache
	promptCacheMu.RUnlock()

	if !cached.loadedAt.IsZero() &&
		time.Since(cached.loadedAt) < promptCacheTTL &&
		cached.configHash == configHash &&
		cached.skillsMaxRev.Equal(skillsMaxRev) &&
		cached.subagentsMaxRev.Equal(subagentsMaxRev) &&
		cached.memoryScope == memoryScope &&
		cached.memoryFingerprint == memoryFP &&
		contextFileMTimesEqual(cached.fileMTimes, fileMTimes) {
		return cached.prompt
	}

	prompt := buildSystemPromptUncached(ctx, ws)
	promptCacheMu.Lock()
	promptCache = promptCacheEntry{
		prompt:            prompt,
		loadedAt:          time.Now().UTC(),
		configHash:        configHash,
		skillsMaxRev:      skillsMaxRev,
		subagentsMaxRev:   subagentsMaxRev,
		memoryFingerprint: memoryFP,
		memoryScope:       memoryScope,
		fileMTimes:        fileMTimes,
	}
	promptCacheVer.Add(1)
	promptCacheMu.Unlock()
	return prompt
}

func loadMemoryFactsFingerprint(ctx context.Context, scope string) string {
	if store.Database == nil {
		return ""
	}
	fp, err := store.Database.GetAgentMemoryFactsFingerprint(ctx, scope)
	if err != nil {
		flog.Warn("[chat-agent] load memory facts fingerprint: %v", err)
		return ""
	}
	return fmt.Sprintf("%d:%s:%s", fp.Count, fp.MaxUpdatedAt.UTC().Format(time.RFC3339Nano), fp.ContentHash)
}

func loadInjectableMemoryFacts(ctx context.Context, scope string) []InjectedMemoryFact {
	if store.Database == nil {
		return nil
	}
	rows, err := store.Database.ListInjectableAgentMemoryFacts(ctx, store.AgentMemoryInjectableParams{
		Scope:    scope,
		MaxCount: 30,
		MaxChars: 4000,
	})
	if err != nil {
		flog.Warn("[chat-agent] load injectable memory facts: %v", err)
		return nil
	}
	out := make([]InjectedMemoryFact, 0, len(rows))
	for _, row := range rows {
		out = append(out, InjectedMemoryFact{Key: row.Key, Value: row.Value, Pinned: row.Pinned})
	}
	return out
}

func buildSystemPromptUncached(ctx context.Context, ws coding.Workspace) string {
	cfg := config.App.ChatAgent
	skills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills: %v", err)
		skills = nil
	}
	subagents, err := LoadSubagentsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load subagents: %v", err)
		subagents = nil
	}
	contextFiles := loadContextFiles(ws.Root, cfg.ContextFiles)
	memoryFacts := loadInjectableMemoryFacts(ctx, resolveToolMemoryScope(ctx))
	flog.Debug("[chat-agent] system prompt workspace=%s skills=%d subagents=%d context_files=%d memory_facts=%d",
		ws.Root, len(skills), len(subagents), len(contextFiles), len(memoryFacts))
	return BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt:       cfg.SystemPrompt,
		PromptGuidelines:   cfg.PromptGuidelines,
		AppendSystemPrompt: cfg.AppendSystemPrompt,
		CWD:                ws.Root,
		ContextFiles:       contextFiles,
		Skills:             skills,
		Subagents:          subagents,
		MemoryFacts:        memoryFacts,
	})
}

// SessionSystemPrompt builds the system prompt for one session mode.
func SessionSystemPrompt(ctx context.Context, ws coding.Workspace, mode string) string {
	if mode != ModePlan {
		return CachedSystemPrompt(ctx, ws)
	}
	return buildPlanModeSystemPrompt(ctx, ws)
}

// buildFilteredSystemPrompt builds a system prompt with only the selected skills injected.
func buildFilteredSystemPrompt(ctx context.Context, ws coding.Workspace, skillNames []string) string {
	cfg := config.App.ChatAgent
	allSkills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills: %v", err)
		allSkills = nil
	}
	skills := FilterSkillsByNames(allSkills, skillNames)
	subagents, err := LoadSubagentsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load subagents: %v", err)
		subagents = nil
	}
	contextFiles := loadContextFiles(ws.Root, cfg.ContextFiles)
	return BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt:       cfg.SystemPrompt,
		PromptGuidelines:   cfg.PromptGuidelines,
		AppendSystemPrompt: cfg.AppendSystemPrompt,
		CWD:                ws.Root,
		ContextFiles:       contextFiles,
		Skills:             skills,
		Subagents:          subagents,
		MemoryFacts:        loadInjectableMemoryFacts(ctx, resolveToolMemoryScope(ctx)),
	})
}

func buildPlanModeSystemPrompt(ctx context.Context, ws coding.Workspace) string {
	cfg := config.App.ChatAgent
	skills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills: %v", err)
		skills = nil
	}
	subagents, err := LoadSubagentsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load subagents: %v", err)
		subagents = nil
	}
	contextFiles := loadContextFiles(ws.Root, cfg.ContextFiles)
	return BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt:       cfg.SystemPrompt,
		PromptGuidelines:   append([]string(nil), cfg.PromptGuidelines...),
		AppendSystemPrompt: cfg.AppendSystemPrompt,
		CWD:                ws.Root,
		ContextFiles:       contextFiles,
		Skills:             skills,
		Subagents:          subagents,
		SelectedTools:      ReadOnlyToolNames(),
		Mode:               ModePlan,
		MemoryFacts:        loadInjectableMemoryFacts(ctx, resolveToolMemoryScope(ctx)),
	})
}

func promptConfigHash(workspaceRoot string) string {
	cfg := config.App.ChatAgent
	language := config.App.Flowbot.Language
	if language == "" {
		language = "English"
	}
	parts := []string{
		workspaceRoot,
		cfg.SystemPrompt,
		cfg.AppendSystemPrompt,
		strings.Join(cfg.PromptGuidelines, "\n"),
		strings.Join(cfg.ContextFiles, "\n"),
		language,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func collectContextFileMTimes(cwd string, explicit []string) map[string]time.Time {
	names := explicit
	if len(names) == 0 {
		names = []string{"AGENTS.md", "README.md"}
	}
	mtimes := make(map[string]time.Time, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, name)
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mtimes[path] = info.ModTime().UTC()
	}
	return mtimes
}

func contextFileMTimesEqual(left, right map[string]time.Time) bool {
	if len(left) != len(right) {
		return false
	}
	for path, mtime := range left {
		other, ok := right[path]
		if !ok || !other.Equal(mtime) {
			return false
		}
	}
	return true
}

func loadSkillsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	if store.Database == nil {
		return time.Time{}, nil
	}
	rev, err := store.Database.GetAgentSkillsMaxUpdatedAt(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("skills max updated_at: %w", err)
	}
	return rev.UTC(), nil
}

func loadSubagentsMaxUpdatedAt(ctx context.Context) (time.Time, error) {
	if store.Database == nil {
		return time.Time{}, nil
	}
	rev, err := store.Database.GetAgentSubagentsMaxUpdatedAt(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("subagents max updated_at: %w", err)
	}
	return rev.UTC(), nil
}
