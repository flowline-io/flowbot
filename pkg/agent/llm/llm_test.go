package llm_test

import (
	"os"
	"testing"

	"github.com/flowline-io/flowbot/pkg/config"
)

func init() {
	config.App.Models = []config.Model{
		{Provider: "openai", ModelNames: []string{"gpt-5.5-instant"}, ApiKey: "sk-test"},
		{Provider: "openai_compatible", ModelNames: []string{"local-model"}, ApiKey: "", BaseUrl: "http://localhost:11434/v1"},
		{Provider: "gemini", ModelNames: []string{"gemini-3.1-pro"}, ApiKey: "test-gemini-key"},
		{Provider: "anthropic", ModelNames: []string{"claude-opus-4.7"}, ApiKey: "test-anthropic-key"},
		{Provider: "anthropic", ModelNames: []string{"claude-proxy"}, ApiKey: "test-anthropic-key", BaseUrl: "http://localhost:8080/v1"},
		{Provider: "openai", ModelNames: []string{"gpt-5.5"}, ApiKey: "sk-test"},
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
