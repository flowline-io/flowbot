package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestPublishFinalUsage(t *testing.T) {
	tests := []struct {
		name          string
		messages      []any
		contextWindow int
		wantTotal     int
		wantWindow    int
		wantPercent   float64
	}{
		{
			name: "computes percent from llm usage",
			messages: []any{
				msg.AssistantMessage{
					Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}},
					Usage: &msg.Usage{PromptTokens: 3000, CompletionTokens: 1016, TotalTokens: 4016},
				},
			},
			contextWindow: 128000,
			wantTotal:     4016,
			wantWindow:    128000,
			wantPercent:   3.1375,
		},
		{
			name:          "skips empty usage",
			messages:      []any{msg.AssistantMessage{Parts: []msg.ContentPart{msg.TextPart{Text: "hi"}}}},
			contextWindow: 128000,
		},
		{
			name: "sums multiple assistant usage blocks",
			messages: []any{
				msg.AssistantMessage{Usage: &msg.Usage{TotalTokens: 1000}},
				msg.AssistantMessage{Usage: &msg.Usage{TotalTokens: 500}},
			},
			contextWindow: 10000,
			wantTotal:     1500,
			wantWindow:    10000,
			wantPercent:   15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := NewChannelPublisher(4)
			publishFinalUsage(pub, tt.messages, tt.contextWindow)
			var got StreamEvent
			select {
			case got = <-pub.Events():
			default:
			}
			if tt.wantTotal == 0 {
				assert.Empty(t, got.Type)
				return
			}
			assert.Equal(t, EventTypeUsage, got.Type)
			assert.Equal(t, tt.wantTotal, got.TotalTokens)
			assert.Equal(t, tt.wantWindow, got.ContextWindow)
			assert.InDelta(t, tt.wantPercent, got.ContextPercent, 0.0001)
		})
	}
}
