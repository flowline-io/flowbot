// Package memos implements the Memos adapter for the memo capability.
package memos

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	memosvc "github.com/flowline-io/flowbot/pkg/ability/memo"
	provider "github.com/flowline-io/flowbot/pkg/providers/memos"
)

// fakeClient implements the client interface for testing.
type fakeClient struct {
	createResp        *provider.Memo
	createErr         error
	getResp           *provider.Memo
	getErr            error
	listResp          *provider.ListMemosResponse
	listErr           error
	updateResp        *provider.Memo
	updateErr         error
	deleteErr         error
	getCurrentUser    *provider.User
	getCurrentUserErr error
	listRawItems      []map[string]any
	listRawCursor     string
	listRawErr        error
}

func (f *fakeClient) CreateMemo(_ context.Context, _, _ string) (*provider.Memo, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) GetMemo(_ context.Context, _ string) (*provider.Memo, error) {
	return f.getResp, f.getErr
}

func (f *fakeClient) ListMemos(_ context.Context, _ provider.ListMemosParams) (*provider.ListMemosResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeClient) UpdateMemo(_ context.Context, _, _, _ string, _ *bool, _ []string) (*provider.Memo, error) {
	return f.updateResp, f.updateErr
}

func (f *fakeClient) DeleteMemo(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeClient) GetCurrentUser(_ context.Context) (*provider.User, error) {
	return f.getCurrentUser, f.getCurrentUserErr
}

func (f *fakeClient) ListRawEvents(_ context.Context, _ string) ([]map[string]any, string, error) {
	return f.listRawItems, f.listRawCursor, f.listRawErr
}

var _ client = (*fakeClient)(nil)

var _ memosvc.Service = (*Adapter)(nil)

func TestAdapter_List(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name: "success with items",
			client: &fakeClient{
				listResp: &provider.ListMemosResponse{
					Memos: []provider.Memo{
						{Name: "memos/1", Content: "First memo", Visibility: "PRIVATE"},
						{Name: "memos/2", Content: "Second memo", Visibility: "PUBLIC"},
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "empty results",
			client: &fakeClient{
				listResp: &provider.ListMemosResponse{Memos: []provider.Memo{}},
			},
			wantLen: 0,
		},
		{
			name:    "provider error",
			client:  &fakeClient{listErr: assert.AnError},
			wantErr: true,
		},
		{
			name: "has more with next page token",
			client: &fakeClient{
				listResp: &provider.ListMemosResponse{
					Memos:         []provider.Memo{{Name: "memos/1", CreateTime: &now}},
					NextPageToken: "token-123",
				},
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.List(context.Background(), nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestAdapter_Get(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		client  *fakeClient
		nameArg string
		wantErr bool
	}{
		{
			name: "success",
			client: &fakeClient{
				getResp: &provider.Memo{Name: "memos/1", Content: "Hello", CreateTime: &now},
			},
			nameArg: "memos/1",
		},
		{
			name:    "empty name returns error",
			client:  &fakeClient{},
			nameArg: "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: assert.AnError},
			nameArg: "memos/1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Get(context.Background(), tt.nameArg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, tt.nameArg, item.Name)
		})
	}
}

func TestAdapter_Create(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		client     *fakeClient
		content    string
		visibility string
		wantErr    bool
	}{
		{
			name: "success with content and visibility",
			client: &fakeClient{
				createResp: &provider.Memo{Name: "memos/1", Content: "Hello", Visibility: "PUBLIC", CreateTime: &now},
			},
			content:    "Hello",
			visibility: "PUBLIC",
		},
		{
			name: "success with empty visibility",
			client: &fakeClient{
				createResp: &provider.Memo{Name: "memos/1", Content: "Hello", Visibility: "PRIVATE", CreateTime: &now},
			},
			content: "Hello",
		},
		{
			name:    "empty content returns error",
			client:  &fakeClient{},
			content: "",
			wantErr: true,
		},
		{
			name: "provider error",
			client: &fakeClient{
				createErr: assert.AnError,
			},
			content: "Hello",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Create(context.Background(), tt.content, tt.visibility)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, "memos/1", item.Name)
		})
	}
}

func TestAdapter_Update(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		client  *fakeClient
		nameArg string
		data    map[string]any
		wantErr bool
	}{
		{
			name: "success update content",
			client: &fakeClient{
				updateResp: &provider.Memo{Name: "memos/1", Content: "Updated", CreateTime: &now},
			},
			nameArg: "memos/1",
			data:    map[string]any{"content": "Updated"},
		},
		{
			name: "success update pinned",
			client: &fakeClient{
				updateResp: &provider.Memo{Name: "memos/1", Pinned: true, CreateTime: &now},
			},
			nameArg: "memos/1",
			data:    map[string]any{"pinned": true},
		},
		{
			name:    "empty name returns error",
			client:  &fakeClient{},
			nameArg: "",
			data:    map[string]any{"content": "test"},
			wantErr: true,
		},
		{
			name: "provider error",
			client: &fakeClient{
				updateErr: assert.AnError,
			},
			nameArg: "memos/1",
			data:    map[string]any{"content": "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Update(context.Background(), tt.nameArg, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
		})
	}
}

func TestAdapter_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		nameArg string
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{},
			nameArg: "memos/1",
		},
		{
			name:    "empty name returns error",
			client:  &fakeClient{},
			nameArg: "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{deleteErr: assert.AnError},
			nameArg: "memos/1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			err := a.Delete(context.Background(), tt.nameArg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAdapter_HealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		want    bool
		wantErr bool
	}{
		{
			name:   "healthy when user exists",
			client: &fakeClient{getCurrentUser: &provider.User{Name: "users/1", Username: "admin"}},
			want:   true,
		},
		{
			name:   "healthy with minimal user",
			client: &fakeClient{getCurrentUser: &provider.User{Name: "users/2"}},
			want:   true,
		},
		{
			name:   "unhealthy on provider error",
			client: &fakeClient{getCurrentUserErr: assert.AnError},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			ok, err := a.HealthCheck(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, ok)
		})
	}
}

func TestAdapter_ListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name: "success with items",
			client: &fakeClient{
				listRawItems: []map[string]any{
					{"name": "memos/1", "content": "test"},
					{"name": "memos/2", "content": "test2"},
				},
				listRawCursor: "next",
			},
			wantLen: 2,
		},
		{
			name: "empty results",
			client: &fakeClient{
				listRawItems:  []map[string]any{},
				listRawCursor: "",
			},
			wantLen: 0,
		},
		{
			name:    "provider error",
			client:  &fakeClient{listRawErr: assert.AnError},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			items, cursor, err := a.ListRawEvents(context.Background(), "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			if tt.wantLen > 0 {
				assert.NotEmpty(t, cursor)
			} else {
				assert.Empty(t, cursor)
			}
		})
	}
}

func TestAdapter_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := NewWithClient(&fakeClient{})

	t.Run("List", func(t *testing.T) {
		_, err := a.List(ctx, nil)
		require.Error(t, err)
	})
	t.Run("Get", func(t *testing.T) {
		_, err := a.Get(ctx, "memos/1")
		require.Error(t, err)
	})
	t.Run("Create", func(t *testing.T) {
		_, err := a.Create(ctx, "content", "")
		require.Error(t, err)
	})
	t.Run("Update", func(t *testing.T) {
		_, err := a.Update(ctx, "memos/1", map[string]any{"content": "test"})
		require.Error(t, err)
	})
	t.Run("Delete", func(t *testing.T) {
		err := a.Delete(ctx, "memos/1")
		require.Error(t, err)
	})
	t.Run("HealthCheck", func(t *testing.T) {
		_, err := a.HealthCheck(ctx)
		require.Error(t, err)
	})
	t.Run("ListRawEvents", func(t *testing.T) {
		_, _, err := a.ListRawEvents(ctx, "")
		require.Error(t, err)
	})
}
