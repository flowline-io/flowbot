package example

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

type fakePollerService struct {
	items  []any
	cursor string
	err    error
}

func (*fakePollerService) GetItem(_ context.Context, _ string) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) ListItems(_ context.Context, _ *ListQuery) (*ability.ListResult[ability.Host], error) {
	return nil, nil
}
func (*fakePollerService) CreateItem(_ context.Context, _ string, _ types.KV) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) UpdateItem(_ context.Context, _ string, _ map[string]any) (*ability.Host, error) {
	return nil, nil
}
func (*fakePollerService) DeleteItem(_ context.Context, _ string) error { return nil }
func (*fakePollerService) HealthCheck(_ context.Context) (bool, error)  { return true, nil }
func (f *fakePollerService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return f.items, f.cursor, f.err
}

func TestExamplePoller_ResourceName(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, "example/events", p.ResourceName())
}

func TestExamplePoller_DefaultInterval(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, 60*time.Second, p.DefaultInterval())
}

func TestExamplePoller_CursorField(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	assert.Equal(t, "cursor", p.CursorField())
}

func TestExamplePoller_DiffKey(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	tests := []struct {
		name string
		item any
		want string
	}{
		{name: "map with id field", item: map[string]any{"id": "abc-123"}, want: "abc-123"},
		{name: "map without id field", item: map[string]any{"key": "val"}, want: "map[key:val]"},
		{name: "string item", item: "plain-string", want: "plain-string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := p.DiffKey(tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExamplePoller_ContentHash(t *testing.T) {
	t.Parallel()
	p := NewExamplePoller(&fakePollerService{})
	tests := []struct {
		name string
		a    any
		b    any
		same bool
	}{
		{name: "same items produce same hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "1"}, same: true},
		{name: "different items produce different hash", a: map[string]any{"id": "1"}, b: map[string]any{"id": "2"}, same: false},
		{name: "hash is non-empty", a: map[string]any{"id": "x"}, same: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash := p.ContentHash(tt.a)
			assert.NotEmpty(t, hash)
			if tt.same && tt.name == "same items produce same hash" {
				assert.Equal(t, hash, p.ContentHash(tt.b))
			}
			if !tt.same && tt.name == "different items produce different hash" {
				assert.NotEqual(t, hash, p.ContentHash(tt.b))
			}
		})
	}
}

func TestExamplePoller_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		svc        *fakePollerService
		cursor     string
		wantItems  int
		wantCursor string
		wantMore   bool
		wantErr    bool
	}{
		{
			name:       "returns items with no cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}},
			wantItems:  1,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:       "returns items with next cursor",
			svc:        &fakePollerService{items: []any{map[string]any{"id": "1"}}, cursor: "next-page"},
			wantItems:  1,
			wantCursor: "next-page",
			wantMore:   true,
			wantErr:    false,
		},
		{
			name:       "empty result",
			svc:        &fakePollerService{items: []any{}},
			wantItems:  0,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewExamplePoller(tt.svc)
			result, err := p.List(context.Background(), tt.cursor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantItems)
			assert.Equal(t, tt.wantCursor, result.NextCursor)
			assert.Equal(t, tt.wantMore, result.HasMore)
		})
	}
}
