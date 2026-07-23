package model_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantOK    bool
		wantName  string
		wantCtx   int
		wantOut   int
		wantFeats int
	}{
		{
			name:      "known deepseek pro model",
			id:        "deepseek-v4-pro",
			wantOK:    true,
			wantName:  "DeepSeek V4 Pro",
			wantCtx:   1_048_576,
			wantOut:   384_000,
			wantFeats: 5,
		},
		{
			name:      "known deepseek flash model",
			id:        "deepseek-v4-flash",
			wantOK:    true,
			wantName:  "DeepSeek V4 Flash",
			wantCtx:   1_048_576,
			wantOut:   384_000,
			wantFeats: 5,
		},
		{
			name:      "known gpt codex model",
			id:        "gpt-5.3-codex",
			wantOK:    true,
			wantName:  "GPT-5.3-Codex",
			wantCtx:   400_000,
			wantOut:   128_000,
			wantFeats: 6,
		},
		{
			name:      "known claude opus model",
			id:        "claude-opus-4.8",
			wantOK:    true,
			wantName:  "Claude Opus 4.8",
			wantCtx:   1_000_000,
			wantOut:   128_000,
			wantFeats: 7,
		},
		{
			name:      "known claude sonnet model",
			id:        "claude-sonnet-4.6",
			wantOK:    true,
			wantName:  "Claude Sonnet 4.6",
			wantCtx:   1_000_000,
			wantOut:   128_000,
			wantFeats: 6,
		},
		{
			name:      "known qwen plus model",
			id:        "qwen3.7-plus",
			wantOK:    true,
			wantName:  "Qwen3.7 Plus",
			wantCtx:   1_000_000,
			wantOut:   65_536,
			wantFeats: 6,
		},
		{
			name:      "known qwen max model",
			id:        "qwen3.7-max",
			wantOK:    true,
			wantName:  "Qwen3.7 Max",
			wantCtx:   1_000_000,
			wantOut:   65_536,
			wantFeats: 5,
		},
		{
			name:      "known gpt pro model",
			id:        "gpt-5.5-pro",
			wantOK:    true,
			wantName:  "GPT-5.5 Pro",
			wantCtx:   1_050_000,
			wantOut:   128_000,
			wantFeats: 7,
		},
		{
			name:      "known grok 4.5 model",
			id:        "grok-4.5",
			wantOK:    true,
			wantName:  "Grok 4.5",
			wantCtx:   256_000,
			wantOut:   128_000,
			wantFeats: 5,
		},
		{
			name:      "known mimo v2.5 model",
			id:        "mimo-v2.5",
			wantOK:    true,
			wantName:  "MiMo V2.5",
			wantCtx:   1_048_576,
			wantOut:   128_000,
			wantFeats: 8,
		},
		{
			name:      "known mimo v2.5 pro model",
			id:        "mimo-v2.5-pro",
			wantOK:    true,
			wantName:  "MiMo V2.5 Pro",
			wantCtx:   1_048_576,
			wantOut:   128_000,
			wantFeats: 8,
		},
		{
			name:   "unknown model",
			id:     "missing-model",
			wantOK: false,
		},
		{
			name:      "empty id",
			id:        "",
			wantOK:    false,
			wantCtx:   0,
			wantOut:   0,
			wantFeats: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			meta, ok := model.Lookup(tt.id)
			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				return
			}
			assert.Equal(t, tt.wantName, meta.Name)
			assert.Equal(t, tt.wantCtx, meta.ContextLength)
			assert.Equal(t, tt.wantOut, meta.MaxOutput)
			assert.Len(t, meta.Features, tt.wantFeats)
		})
	}
}

func TestContextWindowFor(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      int
	}{
		{name: "catalog pro model", modelName: "deepseek-v4-pro", want: 1_048_576},
		{name: "catalog flash model", modelName: "deepseek-v4-flash", want: 1_048_576},
		{name: "catalog codex model", modelName: "gpt-5.3-codex", want: 400_000},
		{name: "catalog claude opus model", modelName: "claude-opus-4.8", want: 1_000_000},
		{name: "catalog claude sonnet model", modelName: "claude-sonnet-4.6", want: 1_000_000},
		{name: "catalog qwen plus model", modelName: "qwen3.7-plus", want: 1_000_000},
		{name: "catalog qwen max model", modelName: "qwen3.7-max", want: 1_000_000},
		{name: "catalog gpt pro model", modelName: "gpt-5.5-pro", want: 1_050_000},
		{name: "catalog grok 4.5 model", modelName: "grok-4.5", want: 256_000},
		{name: "catalog mimo v2.5 model", modelName: "mimo-v2.5", want: 1_048_576},
		{name: "catalog mimo v2.5 pro model", modelName: "mimo-v2.5-pro", want: 1_048_576},
		{name: "unknown fallback", modelName: "fake-model", want: model.DefaultContextWindow},
		{name: "empty name fallback", modelName: "", want: model.DefaultContextWindow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, model.ContextWindowFor(tt.modelName))
		})
	}
}

func TestMaxContextWindow(t *testing.T) {
	tests := []struct {
		name       string
		modelNames []string
		want       int
	}{
		{
			name:       "returns largest window",
			modelNames: []string{"fake-model", "deepseek-v4-pro"},
			want:       1_048_576,
		},
		{
			name:       "falls back when names empty",
			modelNames: nil,
			want:       model.DefaultContextWindow,
		},
		{
			name:       "unknown models use default",
			modelNames: []string{"missing-a", "missing-b"},
			want:       model.DefaultContextWindow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, model.MaxContextWindow(tt.modelNames...))
		})
	}
}

func TestHasFeature(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		feature   model.Feature
		want      bool
	}{
		{name: "known feature", modelName: "deepseek-v4-pro", feature: model.CapFunctionCall, want: true},
		{name: "image input on codex", modelName: "gpt-5.3-codex", feature: model.ModalityImageIn, want: true},
		{name: "file input on claude opus", modelName: "claude-opus-4.8", feature: model.ModalityFileIn, want: true},
		{name: "audio input on mimo v2.5", modelName: "mimo-v2.5", feature: model.ModalityAudioIn, want: true},
		{name: "video input on mimo v2.5 pro", modelName: "mimo-v2.5-pro", feature: model.ModalityVideoIn, want: true},
		{name: "unknown model", modelName: "fake-model", feature: model.CapChat, want: false},
		{name: "missing feature on known model", modelName: "deepseek-v4-pro", feature: model.Feature("CapVision"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, model.HasFeature(tt.modelName, tt.feature))
		})
	}
}

func TestRegisterTestMetadata(t *testing.T) {
	model.RegisterTestMetadata(t, model.Metadata{
		ID:            "test-model",
		Name:          "Test Model",
		ContextLength: 100_000,
	})

	meta, ok := model.Lookup("test-model")
	require.True(t, ok)
	assert.Equal(t, 100_000, meta.ContextLength)
	assert.Equal(t, 100_000, model.ContextWindowFor("test-model"))
}
