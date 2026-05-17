package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
)

func TestNewProvider_OpenAI(t *testing.T) {
	t.Parallel()

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderOpenAI,
		ApiKey:   "sk-test",
	})
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderOpenAI, p.Name())
}

func TestNewProvider_OpenAICompatible(t *testing.T) {
	t.Parallel()

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderOpenAICompatible,
		ApiKey:   "test-key",
		BaseUrl:  "http://localhost:11434/v1",
	})
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderOpenAI, p.Name())
}

func TestNewProvider_Gemini(t *testing.T) {
	t.Parallel()

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderGemini,
		ApiKey:   "test-gemini-key",
	})
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderGemini, p.Name())
}

func TestNewProvider_Anthropic(t *testing.T) {
	t.Parallel()

	p, err := llm.NewProvider(config.Model{
		Provider: llm.ProviderAnthropic,
		ApiKey:   "test-anthropic-key",
	})
	require.NoError(t, err)
	assert.Equal(t, llm.ProviderAnthropic, p.Name())
}

func TestNewProvider_Unknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerType string
	}{
		{
			name:         "empty provider string",
			providerType: "",
		},
		{
			name:         "unknown provider value",
			providerType: "unknown",
		},
		{
			name:         "misspelled openai",
			providerType: "open-ai",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := llm.NewProvider(config.Model{
				Provider: tt.providerType,
				ApiKey:   "test-key",
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, llm.ErrUnknownProvider)
		})
	}
}
