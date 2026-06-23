package chatagent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
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
				sessionTitleGenWG.Add(1)
				go func() {
					defer sessionTitleGenWG.Done()
					generateSessionTitleAsync("sess", "hello", "hi", "fake", func(context.Context, string) (llms.Model, string, error) {
						return nil, "fake", nil
					}, generateSessionTitleLLM)
				}()
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
					sessionTitleGenWG.Add(1)
					go func() {
						defer sessionTitleGenWG.Done()
						generateSessionTitleAsync("sess", "hello", "hi", "fake", func(context.Context, string) (llms.Model, string, error) {
							return nil, "fake", nil
						}, generateSessionTitleLLM)
					}()
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
				sessionTitleGenWG.Add(1)
				go func() {
					defer sessionTitleGenWG.Done()
					generateSessionTitleAsync("sess", "hello", "world", "fake-model", resolver, generateSessionTitleLLM)
				}()
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
				sessionTitleGenWG.Add(1)
				go func() {
					defer sessionTitleGenWG.Done()
					generateSessionTitleAsync("sess", "edit target file", "plan reply", "fake-model", sessionTitleModel, generateSessionTitleLLM)
				}()
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
