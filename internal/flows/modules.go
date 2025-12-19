package flows

import (
	"github.com/flowline-io/flowbot/internal/store"
	"go.uber.org/fx"
)

// Modules wires Flow-related providers.
var Modules = fx.Options(
	fx.Provide(
		func() RuleRegistry {
			return NewChatbotRuleRegistry()
		},
		func() TemplateRenderer {
			return NewSimpleTemplateRenderer()
		},
		func(storeAdapter store.Adapter, reg RuleRegistry, renderer TemplateRenderer) *Engine {
			return NewEngine(storeAdapter, reg, renderer)
		},
		func(storeAdapter store.Adapter) *RateLimiter {
			return NewRateLimiter(storeAdapter)
		},
		func(engine *Engine, storeAdapter store.Adapter) (*QueueManager, error) {
			return NewQueueManager(storeAdapter, engine)
		},
		func(storeAdapter store.Adapter, queue *QueueManager, reg RuleRegistry) *Poller {
			return NewPoller(storeAdapter, queue, reg)
		},
		func(engine *Engine, rateLimiter *RateLimiter, storeAdapter store.Adapter, queue *QueueManager, reg RuleRegistry) *API {
			return NewAPI(engine, rateLimiter, storeAdapter, queue, reg)
		},
	),
)
