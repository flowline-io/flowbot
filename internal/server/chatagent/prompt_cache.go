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
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const promptCacheTTL = 60 * time.Second

type promptCacheEntry struct {
	prompt       string
	loadedAt     time.Time
	configHash   string
	skillsMaxRev time.Time
	fileMTimes   map[string]time.Time
}

var (
	promptCacheMu sync.RWMutex
	promptCache   promptCacheEntry
)

// ResetPromptCacheForTest clears the in-process system prompt cache.
func ResetPromptCacheForTest() {
	promptCacheMu.Lock()
	defer promptCacheMu.Unlock()
	promptCache = promptCacheEntry{}
}

// PromptCacheVersion returns a monotonic version token for prompt cache invalidation.
func PromptCacheVersion() uint64 {
	promptCacheMu.RLock()
	defer promptCacheMu.RUnlock()
	if promptCache.loadedAt.IsZero() {
		return 0
	}
	return uint64(promptCache.loadedAt.UnixNano())
}

// CachedSystemPrompt returns the chat assistant system prompt, reusing a process cache when inputs are unchanged.
func CachedSystemPrompt(ctx context.Context, ws coding.Workspace) string {
	configHash := promptConfigHash(ws.Root)
	fileMTimes := collectContextFileMTimes(ws.Root, config.App.ChatAgent.ContextFiles)
	skillsMaxRev, err := loadSkillsMaxUpdatedAt(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills revision: %v", err)
	}

	promptCacheMu.RLock()
	cached := promptCache
	promptCacheMu.RUnlock()

	if !cached.loadedAt.IsZero() &&
		time.Since(cached.loadedAt) < promptCacheTTL &&
		cached.configHash == configHash &&
		cached.skillsMaxRev.Equal(skillsMaxRev) &&
		contextFileMTimesEqual(cached.fileMTimes, fileMTimes) {
		return cached.prompt
	}

	prompt := buildSystemPromptUncached(ctx, ws)
	promptCacheMu.Lock()
	promptCache = promptCacheEntry{
		prompt:       prompt,
		loadedAt:     time.Now().UTC(),
		configHash:   configHash,
		skillsMaxRev: skillsMaxRev,
		fileMTimes:   fileMTimes,
	}
	promptCacheMu.Unlock()
	return prompt
}

func buildSystemPromptUncached(ctx context.Context, ws coding.Workspace) string {
	cfg := config.App.ChatAgent
	skills, err := LoadSkillsFromStore(ctx)
	if err != nil {
		flog.Warn("[chat-agent] load skills: %v", err)
		skills = nil
	}
	contextFiles := loadContextFiles(ws.Root, cfg.ContextFiles)
	flog.Debug("[chat-agent] system prompt workspace=%s skills=%d context_files=%d",
		ws.Root, len(skills), len(contextFiles))
	return BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt:       cfg.SystemPrompt,
		PromptGuidelines:   cfg.PromptGuidelines,
		AppendSystemPrompt: cfg.AppendSystemPrompt,
		CWD:                ws.Root,
		ContextFiles:       contextFiles,
		Skills:             skills,
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
