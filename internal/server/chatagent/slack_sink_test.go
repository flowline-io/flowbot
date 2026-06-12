package chatagent

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/platforms"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStreamAdapter struct{}

func (*mockStreamAdapter) MessageConvert(data any) protocol.Message {
	return platforms.MessageConvert(data)
}

func (*mockStreamAdapter) EventConvert(_ any) protocol.Event {
	return protocol.Event{}
}

type mockStreamAction struct {
	updateCalls int
	lastTopic   string
	lastMsgID   string
	lastText    string
}

func (*mockStreamAction) SendMessage(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (m *mockStreamAction) UpdateMessage(req protocol.Request) protocol.Response {
	m.updateCalls++
	params := types.KV(req.Params)
	m.lastTopic, _ = params.String("topic")
	m.lastMsgID, _ = params.String("message_id")
	message, _ := params.Any("message")
	if content, ok := message.(protocol.Message); ok && len(content) > 0 {
		if text, ok := content[0].Data["text"].(string); ok {
			m.lastText = text
		}
	}
	return protocol.NewSuccessResponse(nil)
}

func (*mockStreamAction) DeleteMessage(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetLatestEvents(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetSupportedActions(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetStatus(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetVersion(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetUserInfo(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) CreateChannel(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetChannelInfo(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) GetChannelList(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) RegisterChannels(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func (*mockStreamAction) RegisterSlashCommands(protocol.Request) protocol.Response {
	return protocol.NewFailedResponse(protocol.ErrUnsupportedAction.New("unsupported"))
}

func TestSlackStreamSink_update(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantLen int
	}{
		{name: "delta update", text: "hello", wantLen: 5},
		{name: "flush final text", text: "done", wantLen: 4},
		{name: "truncates long text", text: strings.Repeat("x", slackTextLimit+10), wantLen: slackTextLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			action := &mockStreamAction{}
			caller := &platforms.Caller{Action: action, Adapter: &mockStreamAdapter{}}
			sink := NewSlackStreamSink(caller, "C1", "ts-1")

			var err error
			if tt.name == "flush final text" {
				err = sink.Flush(context.Background(), tt.text)
			} else {
				err = sink.OnDelta(context.Background(), tt.text)
			}
			require.NoError(t, err)
			assert.Equal(t, 1, action.updateCalls)
			assert.Equal(t, "C1", action.lastTopic)
			assert.Equal(t, "ts-1", action.lastMsgID)
			assert.Len(t, action.lastText, tt.wantLen)
		})
	}
}

func TestTruncateSlackText(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "under limit", in: 10, want: 10},
		{name: "at limit", in: slackTextLimit, want: slackTextLimit},
		{name: "over limit", in: slackTextLimit + 50, want: slackTextLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			text := strings.Repeat("a", tt.in)
			got := truncateSlackText(text)
			assert.Len(t, got, tt.want)
		})
	}
}
