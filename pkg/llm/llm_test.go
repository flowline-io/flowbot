package llm_test

import (
	"os"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
)

func init() {
	config.App.Models = []config.Model{
		{Provider: "openai", ModelNames: []string{"gpt-5.5-instant"}, ApiKey: "sk-test"},
		{Provider: "openai_compatible", ModelNames: []string{"local-model"}, ApiKey: "", BaseUrl: "http://localhost:11434/v1"},
		{Provider: "gemini", ModelNames: []string{"gemini-3.1-pro"}, ApiKey: "test-gemini-key"},
		{Provider: "anthropic", ModelNames: []string{"claude-opus-4.7"}, ApiKey: "test-anthropic-key"},
		{Provider: "openai", ModelNames: []string{"gpt-5.5"}, ApiKey: "sk-test"},
	}
	config.App.Agents = []config.Agent{
		{Name: "agent_active", Model: "gpt-5.5-instant", Enabled: true},
		{Name: "agent_disabled", Model: "gpt-5.5-instant", Enabled: false},
		{Name: "agent_nomodel", Model: "", Enabled: true},
		{Name: "agent_chat", Model: "gpt-5.5-instant", Enabled: true},
	}
}

func TestMain(m *testing.M) {
	llm.RegisterGemini()
	llm.RegisterOpenAI()
	llm.RegisterAnthropic()
	os.Exit(m.Run())
}
