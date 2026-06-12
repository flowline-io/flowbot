package chatagent

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

const slackTextLimit = 40000

// slackStreamSink updates one Slack message via chat.update during an agent run.
type slackStreamSink struct {
	caller    *platforms.Caller
	topic     string
	messageID string
}

// NewSlackStreamSink creates a sink that edits an existing Slack message.
func NewSlackStreamSink(caller *platforms.Caller, topic, messageID string) StreamSink {
	return &slackStreamSink{
		caller:    caller,
		topic:     topic,
		messageID: messageID,
	}
}

// OnDelta pushes a throttled in-progress snapshot to Slack.
func (s *slackStreamSink) OnDelta(ctx context.Context, text string) error {
	_ = ctx
	return s.update(text)
}

// Flush writes the final reply text to Slack.
func (s *slackStreamSink) Flush(ctx context.Context, final string) error {
	_ = ctx
	return s.update(final)
}

func (s *slackStreamSink) update(text string) error {
	if s.caller == nil || s.messageID == "" {
		return fmt.Errorf("slack stream sink: missing caller or message id")
	}
	text = truncateSlackText(text)
	resp := s.caller.Do(protocol.Request{
		Action: protocol.UpdateMessageAction,
		Params: types.KV{
			"topic":      s.topic,
			"message_id": s.messageID,
			"message":    s.caller.Adapter.MessageConvert(types.TextMsg{Text: text}),
		},
	})
	if resp.Status != protocol.Success {
		err := fmt.Errorf("slack update message failed: %+v", resp)
		flog.Warn("[chat-agent] %v", err)
		return err
	}
	return nil
}

func truncateSlackText(text string) string {
	if len(text) <= slackTextLimit {
		return text
	}
	flog.Warn("[chat-agent] slack reply truncated from %d to %d chars", len(text), slackTextLimit)
	return text[:slackTextLimit]
}
