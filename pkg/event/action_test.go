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

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ---------------------------------------------------------------------------
// Mock adapter for store.Adapter
// ---------------------------------------------------------------------------

type mockStore struct {
	store.Adapter
	platformChannelUsersByFlagsFn func(ctx context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error)
	platformsFn                   func(ctx context.Context) ([]*gen.Platform, error)
}

func (m *mockStore) GetPlatformChannelUsersByUserFlags(ctx context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error) {
	if m.platformChannelUsersByFlagsFn != nil {
		return m.platformChannelUsersByFlagsFn(ctx, userFlags)
	}
	return nil, nil
}

func (m *mockStore) GetPlatforms(ctx context.Context) ([]*gen.Platform, error) {
	if m.platformsFn != nil {
		return m.platformsFn(ctx)
	}
	return nil, nil
}

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
// Test helpers
// ---------------------------------------------------------------------------

func testPayload() types.EventPayload {
	msg := types.TextMsg{Text: "hello"}
	src, _ := sonic.Marshal(msg)
	return types.EventPayload{
		Typ: types.TypeOf(msg),
		Src: src,
	}
}

func testPlatforms() []*gen.Platform {
	return []*gen.Platform{
		{ID: 1, Name: "slack"},
		{ID: 2, Name: "discord"},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSendToAll_EmptyPlatformUsers(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return nil, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	err := sendToAll(ctx, testPayload(), nil, pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_EmptyPlatformUserSlice(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return nil, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	err := sendToAll(ctx, testPayload(), []*gen.PlatformUser{}, pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_SinglePlatformUserWithChannels(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	var capturedFlags []string
	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error) {
			capturedFlags = userFlags
			return []*gen.PlatformChannelUser{
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:random"},
			}, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
	require.NoError(t, err)

	assert.Equal(t, []string{"user:slack:U1"}, capturedFlags)
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
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	var capturedFlags []string
	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, userFlags []string) ([]*gen.PlatformChannelUser, error) {
			capturedFlags = userFlags
			return []*gen.PlatformChannelUser{
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
				{UserFlag: "user:discord:D1", ChannelFlag: "ch:main"},
				{UserFlag: "user:discord:D1", ChannelFlag: "ch:dev"},
			}, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 2, Flag: "user:discord:D1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
	require.NoError(t, err)

	assert.Len(t, capturedFlags, 2)
	assert.Contains(t, capturedFlags, "user:slack:U1")
	assert.Contains(t, capturedFlags, "user:discord:D1")

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
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return []*gen.PlatformChannelUser{
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
			}, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 2, Flag: "user:discord:D1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
	require.NoError(t, err)
	// Only slack channels should be published, discord: no channel users
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
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{}

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return []*gen.Platform{{ID: 1, Name: "slack"}}, nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return []*gen.PlatformChannelUser{
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
				{UserFlag: "user:unknown:X1", ChannelFlag: "ch:random"},
			}, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
		{PlatformID: 999, Flag: "user:unknown:X1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
	require.NoError(t, err)
	// Platform 999 has no name, so only slack messages are published
	assert.Len(t, pub.messages, 1)
}

func TestSendToAll_BatchQueryError(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return nil, errors.New("database connection lost")
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platform channel users")
}

func TestSendToAll_PlatformsQueryError(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return nil, errors.New("database offline")
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platforms")
}

func TestSendToAll_PublisherError(t *testing.T) {
	origStore := store.Database
	defer func() { store.Database = origStore }()

	pub := &mockPublisher{
		err: errors.New("publisher offline"),
	}

	ms := &mockStore{
		platformsFn: func(_ context.Context) ([]*gen.Platform, error) {
			return testPlatforms(), nil
		},
		platformChannelUsersByFlagsFn: func(_ context.Context, _ []string) ([]*gen.PlatformChannelUser, error) {
			return []*gen.PlatformChannelUser{
				{UserFlag: "user:slack:U1", ChannelFlag: "ch:general"},
			}, nil
		},
	}
	store.Database = ms

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
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
