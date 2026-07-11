package chatagent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestWaitForSessionTitleGenerationForTest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T) error
	}{
		{
			name: "waits for in-flight generation",
			run: func(t *testing.T) error {
				var started atomic.Bool
				origLLM := generateSessionTitleLLM
				generateSessionTitleLLM = func(ctx context.Context, _, _, _ string, _ sessionTitleModelFunc) (string, error) {
					started.Store(true)
					<-ctx.Done()
					return "", ctx.Err()
				}
				t.Cleanup(func() {
					generateSessionTitleLLM = origLLM
					WaitForSessionTitleGenerationForTest()
				})
				sessionTitleGenWG.Go(func() {
					generateSessionTitleAsync("sess", "hello", "hi", "fake", func(context.Context, string) (llms.Model, string, error) {
						return nil, "fake", nil
					}, generateSessionTitleLLM)
				})
				require.Eventually(t, started.Load, time.Second, 10*time.Millisecond)
				WaitForSessionTitleGenerationForTest()
				return nil
			},
		},
		{
			name: "no-op when nothing pending",
			run: func(_ *testing.T) error {
				WaitForSessionTitleGenerationForTest()
				return nil
			},
		},
		{
			name: "waits for multiple generations",
			run: func(t *testing.T) error {
				origLLM := generateSessionTitleLLM
				generateSessionTitleLLM = func(_ context.Context, _, _, _ string, _ sessionTitleModelFunc) (string, error) {
					time.Sleep(20 * time.Millisecond)
					return "Title", nil
				}
				t.Cleanup(func() {
					generateSessionTitleLLM = origLLM
					WaitForSessionTitleGenerationForTest()
				})
				for range 3 {
					sessionTitleGenWG.Go(func() {
						generateSessionTitleAsync("sess", "hello", "hi", "fake", func(context.Context, string) (llms.Model, string, error) {
							return nil, "fake", nil
						}, generateSessionTitleLLM)
					})
				}
				WaitForSessionTitleGenerationForTest()
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.run(t))
		})
	}
}

func TestSessionTitleInflightDedupes(t *testing.T) {
	tests := []struct {
		name      string
		sessions  []string
		wantCalls int32
	}{
		{name: "same session deduped", sessions: []string{"sess-a", "sess-a", "sess-a"}, wantCalls: 1},
		{name: "different sessions run", sessions: []string{"sess-b", "sess-c"}, wantCalls: 2},
		{name: "mixed overlap", sessions: []string{"sess-d", "sess-d", "sess-e"}, wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LockAppConfigForTest(t)

			origDB := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			origChatAgent := config.App.ChatAgent
			origModels := config.App.Models
			config.App.ChatAgent = config.ChatAgentConfig{ChatModel: "gpt-4o-mini"}
			config.App.Models = []config.Model{{Provider: "openai", ModelNames: []string{"gpt-4o-mini"}}}

			seen := make(map[string]struct{}, len(tt.sessions))
			for _, sessionID := range tt.sessions {
				if _, ok := seen[sessionID]; ok {
					continue
				}
				seen[sessionID] = struct{}{}
				require.NoError(t, store.Database.CreateChatSession(context.Background(), &gen.ChatSession{
					Flag:  sessionID,
					UID:   "user-1",
					State: int(schema.ChatSessionActive),
				}))
			}

			ResetSessionTitleGenerationForTest()
			var calls atomic.Int32
			releaseCh := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(releaseCh) }) }
			origLLM := generateSessionTitleLLM
			sessionTitleLLMMu.Lock()
			generateSessionTitleLLM = func(_ context.Context, _, _, _ string, _ sessionTitleModelFunc) (string, error) {
				calls.Add(1)
				<-releaseCh
				return "Generated title", nil
			}
			sessionTitleLLMMu.Unlock()
			t.Cleanup(func() {
				unblock()
				sessionTitleLLMMu.Lock()
				generateSessionTitleLLM = origLLM
				sessionTitleLLMMu.Unlock()
				WaitForSessionTitleGenerationForTest()
				store.Database = origDB
				config.App.ChatAgent = origChatAgent
				config.App.Models = origModels
			})

			for _, sessionID := range tt.sessions {
				go maybeGenerateSessionTitle(sessionID, "hello", "world")
			}
			require.Eventually(t, func() bool {
				return calls.Load() >= tt.wantCalls
			}, time.Second, 10*time.Millisecond)
			unblock()
			WaitForSessionTitleGenerationForTest()
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestDisableSessionTitleLLMForTest(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "disable uses fallback path",
			run: func(t *testing.T) {
				restore := DisableSessionTitleLLMForTest()
				t.Cleanup(restore)
				title, err := generateSessionTitleLLM(context.Background(), "hello", "world", "fake-model", sessionTitleModel)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "skipped for test")
				assert.Empty(t, title)
			},
		},
		{
			name: "restore re-enables llm generator",
			run: func(t *testing.T) {
				restore := DisableSessionTitleLLMForTest()
				restore()
				_, err := generateSessionTitleLLM(context.Background(), "hello", "world", "fake-model", func(context.Context, string) (llms.Model, string, error) {
					return nil, "", assert.AnError
				})
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "skipped for test")
			},
		},
		{
			name: "nested disable restore",
			run: func(t *testing.T) {
				outer := DisableSessionTitleLLMForTest()
				inner := DisableSessionTitleLLMForTest()
				inner()
				_, err := generateSessionTitleLLM(context.Background(), "hello", "world", "fake-model", sessionTitleModel)
				assert.Contains(t, err.Error(), "skipped for test")
				outer()
				_, err = generateSessionTitleLLM(context.Background(), "hello", "world", "fake-model", func(context.Context, string) (llms.Model, string, error) {
					return nil, "", assert.AnError
				})
				assert.NotContains(t, err.Error(), "skipped for test")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestSessionTitleModelSeparateFromHarnessFake(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "title resolver does not consume harness fake model",
			run: func(t *testing.T) {
				harnessFake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "harness"})
				titleFake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "Generated title"})
				origHarness := NewModelForTest
				origTitle := sessionTitleModel
				NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
					return harnessFake, "fake-model", nil
				}
				sessionTitleModel = func(_ context.Context, _ string) (llms.Model, string, error) {
					return titleFake, "fake-model", nil
				}
				t.Cleanup(func() {
					NewModelForTest = origHarness
					sessionTitleModel = origTitle
				})

				got, err := generateSessionTitleWithLLM(context.Background(), "hello", "world", "fake-model", sessionTitleModel)
				require.NoError(t, err)
				assert.Equal(t, "Generated title", got)
				assert.Equal(t, 0, harnessFake.Calls())
				assert.Equal(t, 1, titleFake.Calls())
			},
		},
		{
			name: "async title job uses captured resolver",
			run: func(t *testing.T) {
				titleFake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "Async title"})
				origLLM := generateSessionTitleLLM
				generateSessionTitleLLM = generateSessionTitleWithLLM
				resolver := func(_ context.Context, _ string) (llms.Model, string, error) {
					return titleFake, "fake-model", nil
				}
				t.Cleanup(func() {
					generateSessionTitleLLM = origLLM
					WaitForSessionTitleGenerationForTest()
				})
				sessionTitleGenWG.Go(func() {
					generateSessionTitleAsync("sess", "hello", "world", "fake-model", resolver, generateSessionTitleLLM)
				})
				WaitForSessionTitleGenerationForTest()
				assert.Equal(t, 1, titleFake.Calls())
			},
		},
		{
			name: "fallback title avoids model call on resolver error",
			run: func(t *testing.T) {
				origLLM := generateSessionTitleLLM
				generateSessionTitleLLM = func(_ context.Context, _, _, _ string, _ sessionTitleModelFunc) (string, error) {
					return "", assert.AnError
				}
				t.Cleanup(func() {
					generateSessionTitleLLM = origLLM
					WaitForSessionTitleGenerationForTest()
				})
				sessionTitleGenWG.Go(func() {
					generateSessionTitleAsync("sess", "edit target file", "plan reply", "fake-model", sessionTitleModel, generateSessionTitleLLM)
				})
				WaitForSessionTitleGenerationForTest()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		fallback string
	}{
		{name: "plain title unchanged", input: "Deploy flowbot service", want: "Deploy flowbot service"},
		{name: "strips surrounding quotes", input: `"Fix login bug"`, want: "Fix login bug"},
		{name: "collapses newlines", input: "Plan\nfor\nrelease", want: "Plan for release"},
		{name: "truncates long title", input: "This is an extremely long session title that should be trimmed down for display", want: "This is an extremely long session title that should be trimm"},
		{name: "empty falls back to user text", input: "   ", fallback: "How do I configure Redis?", want: "How do I configure Redis?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTitle(tt.input)
			if got == "" && tt.fallback != "" {
				got = fallbackSessionTitle(tt.fallback)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFallbackSessionTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "short message kept", input: "hello world", want: "hello world"},
		{name: "newline collapsed", input: "line one\nline two", want: "line one line two"},
		{name: "long message truncated", input: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcde..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fallbackSessionTitle(tt.input))
		})
	}
}

func TestBuildSessionTitlePrompt(t *testing.T) {
	tests := []struct {
		name     string
		userText string
		reply    string
		wantSub  []string
	}{
		{name: "includes user and assistant", userText: "hi", reply: "hello", wantSub: []string{"User: hi", "Assistant: hello"}},
		{name: "trims whitespace", userText: "  ask  ", reply: "  ok  ", wantSub: []string{"User: ask", "Assistant: ok"}},
		{name: "truncates long reply", userText: "q", reply: string(make([]byte, 600)), wantSub: []string{"..."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSessionTitlePrompt(tt.userText, tt.reply)
			for _, want := range tt.wantSub {
				assert.Contains(t, got, want)
			}
		})
	}
}
