package trilium

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
)

type fakeNotePollerService struct {
	items  []any
	cursor string
	err    error
}

func (*fakeNotePollerService) List(_ context.Context, _ *notesvc.ListQuery) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*fakeNotePollerService) Get(_ context.Context, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*fakeNotePollerService) Create(_ context.Context, _, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*fakeNotePollerService) Update(_ context.Context, _, _, _ string) (*ability.Note, error) {
	return nil, nil
}
func (*fakeNotePollerService) Delete(_ context.Context, _ string) error               { return nil }
func (*fakeNotePollerService) GetContent(_ context.Context, _ string) (string, error) { return "", nil }
func (*fakeNotePollerService) SetContent(_ context.Context, _, _ string) error        { return nil }
func (*fakeNotePollerService) Search(_ context.Context, _ string) (*ability.ListResult[ability.Note], error) {
	return nil, nil
}
func (*fakeNotePollerService) GetAppInfo(_ context.Context) (*ability.Note, error) { return nil, nil }
func (f *fakeNotePollerService) ListRawEvents(_ context.Context, _ string) ([]any, string, error) {
	return f.items, f.cursor, f.err
}

func TestNotePoller_ResourceName(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, "note/events", p.ResourceName())
}

func TestNotePoller_DefaultInterval(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, 120*time.Second, p.DefaultInterval())
}

func TestNotePoller_CursorField(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	assert.Equal(t, "cursor", p.CursorField())
}

func TestNotePoller_DiffKey(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	tests := []struct {
		name string
		item any
		want string
	}{
		{name: "map with noteId field", item: map[string]any{"noteId": "abc-123"}, want: "abc-123"},
		{name: "map without noteId field", item: map[string]any{"key": "val"}, want: "map[key:val]"},
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

func TestNotePoller_ContentHash(t *testing.T) {
	t.Parallel()
	p := NewPollerWithService(&fakeNotePollerService{})
	tests := []struct {
		name string
		a    any
		b    any
		same bool
	}{
		{name: "same items produce same hash", a: map[string]any{"noteId": "1"}, b: map[string]any{"noteId": "1"}, same: true},
		{name: "different items produce different hash", a: map[string]any{"noteId": "1"}, b: map[string]any{"noteId": "2"}, same: false},
		{name: "hash is non-empty", a: map[string]any{"noteId": "x"}, same: true},
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

func TestNotePoller_List(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		svc        *fakeNotePollerService
		cursor     string
		wantItems  int
		wantCursor string
		wantMore   bool
		wantErr    bool
	}{
		{
			name:       "returns items with no cursor",
			svc:        &fakeNotePollerService{items: []any{map[string]any{"noteId": "1"}}},
			wantItems:  1,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:       "returns items with next cursor",
			svc:        &fakeNotePollerService{items: []any{map[string]any{"noteId": "1"}}, cursor: "next-page"},
			wantItems:  1,
			wantCursor: "next-page",
			wantMore:   true,
			wantErr:    false,
		},
		{
			name:       "empty result",
			svc:        &fakeNotePollerService{items: []any{}},
			wantItems:  0,
			wantCursor: "",
			wantMore:   false,
			wantErr:    false,
		},
		{
			name:    "service error",
			svc:     &fakeNotePollerService{err: assert.AnError},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPollerWithService(tt.svc)
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
