package event

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/flowline-io/flowbot/pkg/types"
)

// ---------------------------------------------------------------------------
// Mock publisher for message.Publisher
// ---------------------------------------------------------------------------

type mockPublisher struct {
	mu       sync.Mutex
	messages []*message.Message
	err      error
}

func (m *mockPublisher) Publish(_ string, msgs ...*message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msgs...)
	return m.err
}

func (m *mockPublisher) Close() error {
	return m.err
}

// ---------------------------------------------------------------------------
// Fake MessageDestinations
// ---------------------------------------------------------------------------

type fakeMessageDestinations struct {
	platforms    []*DestinationPlatform
	channelUsers []*DestinationChannelUser

	platformsErr    error
	channelUsersErr error
}

func (*fakeMessageDestinations) GetUserByFlag(context.Context, string) (*DestinationUser, error) {
	return nil, errors.New("not implemented")
}

func (*fakeMessageDestinations) GetPlatformUsersByUserId(context.Context, int64) ([]*DestinationPlatformUser, error) {
	return nil, errors.New("not implemented")
}

func (*fakeMessageDestinations) GetPlatformChannelByFlag(context.Context, string) (*DestinationPlatformChannel, error) {
	return nil, errors.New("not implemented")
}

func (*fakeMessageDestinations) GetPlatform(context.Context, int64) (*DestinationPlatform, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeMessageDestinations) GetPlatforms(context.Context) ([]*DestinationPlatform, error) {
	if f.platformsErr != nil {
		return nil, f.platformsErr
	}
	return f.platforms, nil
}

func (f *fakeMessageDestinations) GetPlatformChannelUsersByUserFlags(context.Context, []string) ([]*DestinationChannelUser, error) {
	if f.channelUsersErr != nil {
		return nil, f.channelUsersErr
	}
	return f.channelUsers, nil
}

func useFakeDestinations(t *testing.T, fake *fakeMessageDestinations) {
	t.Helper()
	orig := GetMessageDestinations()
	SetMessageDestinations(fake)
	t.Cleanup(func() { SetMessageDestinations(orig) })
}

func defaultPlatforms() []*DestinationPlatform {
	return []*DestinationPlatform{
		{ID: 1, Name: "slack"},
		{ID: 2, Name: "discord"},
	}
}

func testPayload() types.EventPayload {
	msg := types.TextMsg{Text: "hello"}
	src, _ := sonic.Marshal(msg)
	return types.EventPayload{
		Typ: types.TypeOf(msg),
		Src: src,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSendToAll_EmptyPlatformUsers(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{platforms: defaultPlatforms()})

	pub := &mockPublisher{}
	err := sendToAll(types.Context{}, testPayload(), nil, GetMessageDestinations(), pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_EmptyPlatformUserSlice(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{platforms: defaultPlatforms()})

	pub := &mockPublisher{}
	err := sendToAll(types.Context{}, testPayload(), []*DestinationPlatformUser{}, GetMessageDestinations(), pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_SinglePlatformUserWithChannels(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms: defaultPlatforms(),
		channelUsers: []*DestinationChannelUser{
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:random"},
		},
	})

	pub := &mockPublisher{}
	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), pub)
	require.NoError(t, err)
	assert.Len(t, pub.messages, 2)

	var topics []string
	for _, msg := range pub.messages {
		var m types.Message
		_ = sonic.Unmarshal(msg.Payload, &m)
		topics = append(topics, m.Topic)
	}
	assert.Contains(t, topics, "ch:general")
	assert.Contains(t, topics, "ch:random")
}

func TestSendToAll_MultiplePlatformUsers(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms: defaultPlatforms(),
		channelUsers: []*DestinationChannelUser{
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
			{UserFlag: "user:discord:D1", ChannelFlag: "ch:main"},
			{UserFlag: "user:discord:D1", ChannelFlag: "ch:dev"},
		},
	})

	pub := &mockPublisher{}
	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 2, Flag: "user:discord:D1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), pub)
	require.NoError(t, err)

	var platforms []string
	for _, msg := range pub.messages {
		var m types.Message
		_ = sonic.Unmarshal(msg.Payload, &m)
		platforms = append(platforms, m.Platform)
	}
	require.Len(t, pub.messages, 3)
	assert.Contains(t, platforms, "slack")
	assert.Contains(t, platforms, "discord")
}

func TestSendToAll_PlatformUserWithNoChannels(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms: defaultPlatforms(),
		channelUsers: []*DestinationChannelUser{
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
		},
	})

	pub := &mockPublisher{}
	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 2, Flag: "user:discord:D1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), pub)
	require.NoError(t, err)

	var platforms []string
	for _, msg := range pub.messages {
		var m types.Message
		_ = sonic.Unmarshal(msg.Payload, &m)
		platforms = append(platforms, m.Platform)
	}
	assert.Len(t, pub.messages, 1)
	assert.Equal(t, "slack", platforms[0])
}

func TestSendToAll_MissingPlatformName(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms: []*DestinationPlatform{{ID: 1, Name: "slack"}},
		channelUsers: []*DestinationChannelUser{
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
			{UserFlag: "user:unknown:X1", ChannelFlag: "ch:random"},
		},
	})

	pub := &mockPublisher{}
	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 999, Flag: "user:unknown:X1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), pub)
	require.NoError(t, err)
	assert.Len(t, pub.messages, 1)
}

func TestSendToAll_BatchQueryError(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms:       defaultPlatforms(),
		channelUsersErr: errors.New("channel users unavailable"),
	})

	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platform channel users")
}

func TestSendToAll_PlatformsQueryError(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platformsErr: errors.New("platforms unavailable"),
	})

	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platforms")
}

func TestSendToAll_PublisherError(t *testing.T) {
	useFakeDestinations(t, &fakeMessageDestinations{
		platforms: defaultPlatforms(),
		channelUsers: []*DestinationChannelUser{
			{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
		},
	})

	pub := &mockPublisher{err: errors.New("publisher offline")}
	platformUsers := []*DestinationPlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(types.Context{}, testPayload(), platformUsers, GetMessageDestinations(), pub)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Existing tests (preserved)
// ---------------------------------------------------------------------------

func TestEventConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "MessageSendEvent", value: types.MessageSendEvent},
		{name: "BotRunEvent", value: types.BotRunEvent},
		{name: "InstructPushEvent", value: types.InstructPushEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.value)
		})
	}
}

func TestActionFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		{name: "SendMessage", fn: SendMessage},
		{name: "BotEventFire", fn: BotEventFire},
		{name: "PublishMessage", fn: PublishMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.fn)
		})
	}
}

func TestPublishWithTracePropagation(t *testing.T) {
	tests := []struct {
		name  string
		topic string
	}{
		{name: "topic alpha propagates publish as receive parent", topic: "alpha.topic"},
		{name: "topic beta propagates publish as receive parent", topic: "beta.topic"},
		{name: "topic gamma propagates publish as receive parent", topic: "gamma.topic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			prevTP := otel.GetTracerProvider()
			prevProp := otel.GetTextMapPropagator()
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			t.Cleanup(func() {
				_ = tp.Shutdown(context.Background())
				otel.SetTracerProvider(prevTP)
				otel.SetTextMapPropagator(prevProp)
			})

			pub := &mockPublisher{}
			parentCtx, parent := tp.Tracer("test").Start(context.Background(), "parent")
			defer parent.End()

			err := publishWith(parentCtx, pub, tt.topic, map[string]string{"k": "v"})
			require.NoError(t, err)
			require.Len(t, pub.messages, 1)

			msg := pub.messages[0]
			require.NotEmpty(t, msg.Metadata.Get("traceparent"))
			assert.Equal(t, tt.topic, msg.Metadata.Get("x-otel-topic"))

			publishName := "event.publish " + tt.topic
			var publishSpanID string
			for _, s := range recorder.Ended() {
				if s.Name() == publishName {
					publishSpanID = s.SpanContext().SpanID().String()
					break
				}
			}
			require.NotEmpty(t, publishSpanID)

			carrier := propagation.MapCarrier{}
			for k, v := range msg.Metadata {
				carrier.Set(k, v)
			}
			extracted := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
			recvName := "event.receive " + tt.topic
			_, recv := tp.Tracer("watermill").Start(extracted, recvName)
			recv.End()

			var recvParent string
			for _, s := range recorder.Ended() {
				if s.Name() == recvName {
					recvParent = s.Parent().SpanID().String()
					break
				}
			}
			assert.Equal(t, publishSpanID, recvParent)
		})
	}
}
