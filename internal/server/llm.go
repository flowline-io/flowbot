package server

import (
	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/pkg/llm"
)

// LLMModules registers all LLM provider backends via fx.
var LLMModules = fx.Options(
	fx.Invoke(
		llm.RegisterGemini,
		llm.RegisterOpenAI,
		llm.RegisterAnthropic,
	),
)
