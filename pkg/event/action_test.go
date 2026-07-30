package event

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/postgres"
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
// Test helpers
// ---------------------------------------------------------------------------

func setupEventTestDB(t *testing.T) {
	t.Helper()
	orig := store.Database
	store.Database = postgres.NewSQLiteTestAdapter(t)
	t.Cleanup(func() { store.Database = orig })
}

func seedTestPlatform(t *testing.T, name string) int64 {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	id, err := store.PlatformStoreFromDB().CreatePlatform(ctx, &gen.Platform{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	return id
}

func seedTestPlatformChannelUser(t *testing.T, platformID int64, userFlag, channelFlag string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	_, err := store.PlatformStoreFromDB().CreatePlatformChannelUser(ctx, &gen.PlatformChannelUser{
		PlatformID:  platformID,
		UserFlag:    userFlag,
		ChannelFlag: channelFlag,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	require.NoError(t, err)
}

func seedTestPlatforms(t *testing.T) (slackID, discordID int64) {
	t.Helper()
	slackID = seedTestPlatform(t, "slack")
	discordID = seedTestPlatform(t, "discord")
	return slackID, discordID
}

func execSQLiteDDL(t *testing.T, client *gen.Client, stmt string) {
	t.Helper()
	db := entClientDB(t, client)
	if _, err := db.Exec(stmt); err != nil {
		t.Fatalf("exec sqlite ddl %q: %v", stmt, err)
	}
}

func entClientDB(t *testing.T, client *gen.Client) *sql.DB {
	t.Helper()
	rv := reflect.ValueOf(client).Elem()
	configField := rv.FieldByName("config")
	config := reflect.NewAt(configField.Type(), unsafe.Pointer(configField.UnsafeAddr())).Elem()
	driverField := config.FieldByName("driver")
	driver := reflect.NewAt(driverField.Type(), unsafe.Pointer(driverField.UnsafeAddr())).Elem()
	switch d := driver.Interface().(type) {
	case *entsql.Driver:
		return d.DB()
	case entsql.Driver:
		return d.DB()
	default:
		t.Fatalf("unexpected ent driver type %T", driver.Interface())
		return nil
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
	setupEventTestDB(t)
	seedTestPlatforms(t)

	pub := &mockPublisher{}

	ctx := types.Context{}
	err := sendToAll(ctx, testPayload(), nil, pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_EmptyPlatformUserSlice(t *testing.T) {
	setupEventTestDB(t)
	seedTestPlatforms(t)

	pub := &mockPublisher{}

	ctx := types.Context{}
	err := sendToAll(ctx, testPayload(), []*gen.PlatformUser{}, pub)
	require.NoError(t, err)
	assert.Empty(t, pub.messages)
}

func TestSendToAll_SinglePlatformUserWithChannels(t *testing.T) {
	setupEventTestDB(t)
	slackID, _ := seedTestPlatforms(t)
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:general")
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:random")

	pub := &mockPublisher{}

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
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
	setupEventTestDB(t)
	slackID, discordID := seedTestPlatforms(t)
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:general")
	seedTestPlatformChannelUser(t, discordID, "user:discord:D1", "ch:main")
	seedTestPlatformChannelUser(t, discordID, "user:discord:D1", "ch:dev")

	pub := &mockPublisher{}

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
		{PlatformID: discordID, Flag: "user:discord:D1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
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
	setupEventTestDB(t)
	slackID, discordID := seedTestPlatforms(t)
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:general")

	pub := &mockPublisher{}

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
		{PlatformID: discordID, Flag: "user:discord:D1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
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
	setupEventTestDB(t)
	slackID := seedTestPlatform(t, "slack")
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:general")
	seedTestPlatformChannelUser(t, slackID, "user:unknown:X1", "ch:random")

	pub := &mockPublisher{}

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
		{PlatformID: 999, Flag: "user:unknown:X1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, pub)
	require.NoError(t, err)
	assert.Len(t, pub.messages, 1)
}

func TestSendToAll_BatchQueryError(t *testing.T) {
	setupEventTestDB(t)
	slackID, _ := seedTestPlatforms(t)
	execSQLiteDDL(t, store.Database.GetClient(), "DROP TABLE platform_channel_users")

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platform channel users")
}

func TestSendToAll_PlatformsQueryError(t *testing.T) {
	setupEventTestDB(t)
	require.NoError(t, store.Database.GetClient().Close())

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: 1, Flag: "user:slack:U1"},
	}

	err := sendToAll(ctx, testPayload(), platformUsers, &mockPublisher{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get platforms")
}

func TestSendToAll_PublisherError(t *testing.T) {
	setupEventTestDB(t)
	slackID, _ := seedTestPlatforms(t)
	seedTestPlatformChannelUser(t, slackID, "user:slack:U1", "ch:general")

	pub := &mockPublisher{
		err: errors.New("publisher offline"),
	}

	ctx := types.Context{}
	platformUsers := []*gen.PlatformUser{
		{PlatformID: slackID, Flag: "user:slack:U1"},
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
