package email

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/email"
)

type fakeClient struct {
	cfg        provider.Config
	sendErr    error
	listMeta   []provider.MessageMeta
	listNext   string
	listErr    error
	searchMeta []provider.MessageMeta
	searchNext string
	searchErr  error
	getMsg     *provider.Message
	getErr     error
	markErr    error
	healthErr  error
	rawItems   []map[string]any
	rawNext    string
	rawErr     error
	lastSend   provider.SendInput
	lastMarkID string
	lastSeen   bool
}

func (f *fakeClient) Send(_ context.Context, in provider.SendInput) error {
	f.lastSend = in
	return f.sendErr
}
func (f *fakeClient) List(_ context.Context, _ string, _ bool, _ int, _ string) ([]provider.MessageMeta, string, error) {
	return f.listMeta, f.listNext, f.listErr
}
func (f *fakeClient) Search(_ context.Context, _ provider.SearchQuery) ([]provider.MessageMeta, string, error) {
	return f.searchMeta, f.searchNext, f.searchErr
}
func (f *fakeClient) Get(_ context.Context, _ string) (*provider.Message, error) {
	return f.getMsg, f.getErr
}
func (f *fakeClient) MarkSeen(_ context.Context, id string, seen bool) error {
	f.lastMarkID = id
	f.lastSeen = seen
	return f.markErr
}
func (f *fakeClient) HealthCheck(_ context.Context) error { return f.healthErr }
func (f *fakeClient) ListRawEvents(_ context.Context, _ string) ([]map[string]any, string, error) {
	return f.rawItems, f.rawNext, f.rawErr
}
func (f *fakeClient) Config() provider.Config { return f.cfg }

var _ client = (*fakeClient)(nil)

func TestAdapter_Send(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      SendInput
		wantErr bool
	}{
		{name: "ok", in: SendInput{To: []string{"a@b.c"}, Subject: "Hi", Text: "body"}},
		{name: "missing to", in: SendInput{Subject: "Hi", Text: "body"}, wantErr: true},
		{name: "missing body", in: SendInput{To: []string{"a@b.c"}, Subject: "Hi"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fc := &fakeClient{}
			err := NewWithClient(fc).Send(context.Background(), tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.in.To, fc.lastSend.To)
			assert.Equal(t, tt.in.Subject, fc.lastSend.Subject)
		})
	}
}

func TestAdapter_ListGetMark(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{
		cfg: provider.Config{UnseenOnly: true},
		listMeta: []provider.MessageMeta{{
			ID: "1:2", Subject: "Hello", From: []string{"a@b.c"}, Date: time.Unix(0, 0).UTC(),
		}},
		listNext: "2",
		getMsg: &provider.Message{
			MessageMeta: provider.MessageMeta{ID: "1:2", Subject: "Hello"},
			Text:        "body",
		},
	}
	svc := NewWithClient(fc)

	list, err := svc.List(context.Background(), ListInput{Page: capability.PageRequest{Limit: 10}})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "1:2", list.Items[0].ID)
	assert.True(t, list.Page.HasMore)
	assert.NotEmpty(t, list.Page.NextCursor)
	assert.NotEqual(t, "2", list.Page.NextCursor) // opaque, not raw UID

	msg, err := svc.Get(context.Background(), "1:2")
	require.NoError(t, err)
	assert.Equal(t, "body", msg.Text)

	require.NoError(t, svc.MarkRead(context.Background(), "1:2"))
	assert.Equal(t, "1:2", fc.lastMarkID)
	assert.True(t, fc.lastSeen)

	require.NoError(t, svc.MarkUnread(context.Background(), "1:2"))
	assert.False(t, fc.lastSeen)
}

func TestAdapter_Health(t *testing.T) {
	t.Parallel()
	ok, err := NewWithClient(&fakeClient{}).HealthCheck(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRegisterCatalog(t *testing.T) {
	t.Parallel()
	spec := CatalogSpec()
	assert.Equal(t, "email", string(spec.Type))
	assert.NotEmpty(t, spec.Ops)
	require.NoError(t, Register("email", nil))
}

func TestPollerDiffKey(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(NewWithClient(&fakeClient{}))
	assert.Equal(t, "email/messages", p.ResourceName())
	assert.Equal(t, "email", p.Capability())
	assert.Equal(t, "9:1", p.DiffKey(map[string]any{"id": "9:1"}))
}

func TestAdapter_MarkEmittedSeenRespectsConfig(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{cfg: provider.Config{MarkSeenAfterEmit: true}}
	svc, ok := NewWithClient(fc).(*Adapter)
	require.True(t, ok)
	require.NoError(t, svc.MarkEmittedSeen(context.Background(), []string{"abc"}))
	assert.Equal(t, "abc", fc.lastMarkID)
	assert.True(t, fc.lastSeen)

	fc2 := &fakeClient{cfg: provider.Config{MarkSeenAfterEmit: false}}
	svc2, ok := NewWithClient(fc2).(*Adapter)
	require.True(t, ok)
	require.NoError(t, svc2.MarkEmittedSeen(context.Background(), []string{"abc"}))
	assert.Empty(t, fc2.lastMarkID)
}
