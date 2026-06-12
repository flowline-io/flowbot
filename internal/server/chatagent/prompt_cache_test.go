package chatagent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedSystemPromptCacheHit(t *testing.T) {
	tests := []struct {
		name        string
		preload     string
		wantPrompt  string
		mutateCache func(root string)
		wantMiss    bool
	}{
		{
			name:       "returns cached prompt when inputs unchanged",
			preload:    "cached prompt",
			wantPrompt: "cached prompt",
		},
		{
			name:       "rebuilds after ttl expiry",
			preload:    "stale prompt",
			wantPrompt: "stale prompt",
			mutateCache: func(root string) {
				promptCacheMu.Lock()
				promptCache.loadedAt = time.Now().UTC().Add(-2 * promptCacheTTL)
				promptCache.configHash = promptConfigHash(root)
				promptCache.fileMTimes = collectContextFileMTimes(root, nil)
				promptCacheMu.Unlock()
			},
			wantMiss: true,
		},
		{
			name:       "rebuilds after config hash change",
			preload:    "old prompt",
			wantPrompt: "old prompt",
			mutateCache: func(root string) {
				promptCacheMu.Lock()
				promptCache.configHash = "stale-hash"
				promptCache.fileMTimes = collectContextFileMTimes(root, nil)
				promptCacheMu.Unlock()
			},
			wantMiss: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetPromptCacheForTest()
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# rules"), 0o644))

			originalPrompt := config.App.ChatAgent.SystemPrompt
			t.Cleanup(func() {
				config.App.ChatAgent.SystemPrompt = originalPrompt
				ResetPromptCacheForTest()
			})
			config.App.ChatAgent.SystemPrompt = ""

			promptCacheMu.Lock()
			promptCache = promptCacheEntry{
				prompt:       tt.preload,
				loadedAt:     time.Now().UTC(),
				configHash:   promptConfigHash(root),
				fileMTimes:   collectContextFileMTimes(root, nil),
				skillsMaxRev: time.Time{},
			}
			promptCacheMu.Unlock()

			if tt.mutateCache != nil {
				tt.mutateCache(root)
			}

			before := PromptCacheVersion()
			got := CachedSystemPrompt(context.Background(), coding.Workspace{Root: root})
			after := PromptCacheVersion()

			if tt.wantMiss {
				assert.NotEqual(t, before, after)
				assert.NotEmpty(t, got)
				return
			}
			assert.Equal(t, tt.wantPrompt, got)
		})
	}
}

func TestPromptConfigHashStable(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func()
		changes bool
	}{
		{
			name:    "same inputs same hash",
			changes: false,
		},
		{
			name: "append prompt changes hash",
			mutate: func() {
				config.App.ChatAgent.AppendSystemPrompt = "extra"
			},
			changes: true,
		},
		{
			name: "language changes hash",
			mutate: func() {
				config.App.Flowbot.Language = "French"
			},
			changes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			originalAppend := config.App.ChatAgent.AppendSystemPrompt
			originalLanguage := config.App.Flowbot.Language
			t.Cleanup(func() {
				config.App.ChatAgent.AppendSystemPrompt = originalAppend
				config.App.Flowbot.Language = originalLanguage
			})

			first := promptConfigHash(root)
			if tt.mutate != nil {
				tt.mutate()
			}
			second := promptConfigHash(root)
			if tt.changes {
				assert.NotEqual(t, first, second)
				return
			}
			assert.Equal(t, first, second)
		})
	}
}

func TestContextFileMTimesEqual(t *testing.T) {
	tests := []struct {
		name  string
		left  map[string]time.Time
		right map[string]time.Time
		want  bool
	}{
		{
			name:  "matching mtimes",
			left:  map[string]time.Time{"/a": time.Unix(1, 0).UTC()},
			right: map[string]time.Time{"/a": time.Unix(1, 0).UTC()},
			want:  true,
		},
		{
			name:  "different path count",
			left:  map[string]time.Time{"/a": time.Unix(1, 0).UTC()},
			right: map[string]time.Time{},
			want:  false,
		},
		{
			name:  "different mtime",
			left:  map[string]time.Time{"/a": time.Unix(1, 0).UTC()},
			right: map[string]time.Time{"/a": time.Unix(2, 0).UTC()},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, contextFileMTimesEqual(tt.left, tt.right))
		})
	}
}

func TestEvictHarnessPool(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		preload   bool
		evictID   string
		wantHit   bool
	}{
		{name: "evicts existing entry", sessionID: "sess-a", preload: true, evictID: "sess-a", wantHit: false},
		{name: "missing entry stays missing", sessionID: "sess-b", preload: false, evictID: "sess-b", wantHit: false},
		{name: "other session remains pooled", sessionID: "sess-c", preload: true, evictID: "other", wantHit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetHarnessPoolForTest()
			if tt.preload {
				harnessPool.Store(tt.sessionID, &pooledHarness{lastUsed: time.Now()})
			}
			EvictHarnessPool(tt.evictID)
			_, ok := harnessPool.Load(tt.sessionID)
			assert.Equal(t, tt.wantHit, ok)
		})
	}
}
