// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	provider "github.com/flowline-io/flowbot/pkg/providers/trilium"
)

// fakeClient implements the client interface for testing.
type fakeClient struct {
	createResp       *provider.NoteWithBranch
	createErr        error
	getResp          *provider.Note
	getErr           error
	patchResp        *provider.Note
	patchErr         error
	deleteErr        error
	searchResp       *provider.SearchResponse
	searchErr        error
	getContentResp   string
	getContentErr    error
	updateContentErr error
	appInfoResp      *provider.AppInfo
	appInfoErr       error
}

func (f *fakeClient) CreateNote(_ context.Context, _ provider.CreateNoteDef) (*provider.NoteWithBranch, error) {
	return f.createResp, f.createErr
}

func (f *fakeClient) GetNote(_ context.Context, _ string) (*provider.Note, error) {
	return f.getResp, f.getErr
}

func (f *fakeClient) PatchNote(_ context.Context, _ string, _ provider.PatchNoteRequest) (*provider.Note, error) {
	return f.patchResp, f.patchErr
}

func (f *fakeClient) DeleteNote(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeClient) SearchNotes(_ context.Context, _ provider.SearchParams) (*provider.SearchResponse, error) {
	return f.searchResp, f.searchErr
}

func (f *fakeClient) GetNoteContent(_ context.Context, _ string) (string, error) {
	return f.getContentResp, f.getContentErr
}

func (f *fakeClient) UpdateNoteContent(_ context.Context, _, _ string) error {
	return f.updateContentErr
}

func (f *fakeClient) GetAppInfo(_ context.Context) (*provider.AppInfo, error) {
	return f.appInfoResp, f.appInfoErr
}

var _ client = (*fakeClient)(nil)

func TestAdapter_List(t *testing.T) {
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
				searchResp: &provider.SearchResponse{
					Results: []provider.Note{
						{NoteID: "n-1", Title: "Note 1", Type: "text"},
						{NoteID: "n-2", Title: "Note 2", Type: "text"},
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "empty results",
			client: &fakeClient{
				searchResp: &provider.SearchResponse{Results: []provider.Note{}},
			},
			wantLen: 0,
		},
		{
			name:    "provider error",
			client:  &fakeClient{searchErr: assert.AnError},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.List(context.Background(), &notesvc.ListQuery{})
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
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{getResp: &provider.Note{NoteID: "n-1", Title: "Test", Type: "text"}},
			id:     "n-1",
		},
		{
			name:    "empty id returns error",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: assert.AnError},
			id:      "n-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Get(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, tt.id, item.ID)
		})
	}
}

func TestAdapter_Create(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		client       *fakeClient
		title        string
		typ          string
		parentNoteID string
		wantErr      bool
	}{
		{
			name: "success with all fields",
			client: &fakeClient{
				createResp: &provider.NoteWithBranch{
					Note:   provider.Note{NoteID: "new-id", Title: "Hello", Type: "text"},
					Branch: provider.Branch{BranchID: "br-1", NoteID: "new-id"},
				},
			},
			title:        "Hello",
			typ:          "text",
			parentNoteID: "root",
		},
		{
			name: "default type text when empty",
			client: &fakeClient{
				createResp: &provider.NoteWithBranch{
					Note:   provider.Note{NoteID: "new-id", Title: "Hello", Type: "text"},
					Branch: provider.Branch{BranchID: "br-1", NoteID: "new-id"},
				},
			},
			title: "Hello",
		},
		{
			name:    "empty title returns error",
			client:  &fakeClient{},
			title:   "",
			wantErr: true,
		},
		{
			name: "provider error",
			client: &fakeClient{
				createErr: assert.AnError,
			},
			title:   "Hello",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Create(context.Background(), tt.title, "content", tt.typ, tt.parentNoteID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, "new-id", item.ID)
		})
	}
}

func TestAdapter_Update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name: "success update title only",
			client: &fakeClient{
				patchResp: &provider.Note{NoteID: "n-1", Title: "Updated", Type: "text"},
				getResp:   &provider.Note{NoteID: "n-1", Title: "Updated", Type: "text"},
			},
			id: "n-1",
		},
		{
			name: "success update title and content",
			client: &fakeClient{
				patchResp: &provider.Note{NoteID: "n-1", Title: "Updated", Type: "text"},
				getResp:   &provider.Note{NoteID: "n-1", Title: "Updated", Type: "text"},
			},
			id: "n-1",
		},
		{
			name:    "empty id returns error",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name: "patch error",
			client: &fakeClient{
				patchErr: assert.AnError,
			},
			id:      "n-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.Update(context.Background(), tt.id, "Updated", "")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, tt.id, item.ID)
		})
	}
}

func TestAdapter_Delete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{},
			id:     "n-1",
		},
		{
			name:    "empty id returns error",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{deleteErr: assert.AnError},
			id:      "n-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			err := a.Delete(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAdapter_GetContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		want    string
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{getContentResp: "<p>Hello World</p>"},
			id:     "n-1",
			want:   "<p>Hello World</p>",
		},
		{
			name:    "empty id returns error",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getContentErr: assert.AnError},
			id:      "n-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			content, err := a.GetContent(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, content)
		})
	}
}

func TestAdapter_SetContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{},
			id:     "n-1",
		},
		{
			name:    "empty id returns error",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{updateContentErr: assert.AnError},
			id:      "n-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			err := a.SetContent(context.Background(), tt.id, "new content")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAdapter_Search(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name: "success with results",
			client: &fakeClient{
				searchResp: &provider.SearchResponse{
					Results: []provider.Note{
						{NoteID: "n-1", Title: "Match 1", Type: "text"},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "empty results",
			client: &fakeClient{
				searchResp: &provider.SearchResponse{Results: []provider.Note{}},
			},
			wantLen: 0,
		},
		{
			name:    "provider error",
			client:  &fakeClient{searchErr: assert.AnError},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.Search(context.Background(), "test query")
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

func TestAdapter_GetAppInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantErr bool
	}{
		{
			name: "success",
			client: &fakeClient{
				appInfoResp: &provider.AppInfo{
					AppVersion:  "0.63.7",
					DBVersion:   1,
					SyncVersion: 1,
				},
			},
		},
		{
			name:    "provider error",
			client:  &fakeClient{appInfoErr: assert.AnError},
			wantErr: true,
		},
		{
			name: "success with instance name",
			client: &fakeClient{
				appInfoResp: &provider.AppInfo{
					AppVersion:   "0.63.7",
					InstanceName: "my-trilium",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			info, err := a.GetAppInfo(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, info)
		})
	}
}

func TestAdapter_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := NewWithClient(&fakeClient{})

	t.Run("List", func(t *testing.T) {
		_, err := a.List(ctx, &notesvc.ListQuery{})
		require.Error(t, err)
	})
	t.Run("Get", func(t *testing.T) {
		_, err := a.Get(ctx, "n-1")
		require.Error(t, err)
	})
	t.Run("Create", func(t *testing.T) {
		_, err := a.Create(ctx, "title", "content", "text", "")
		require.Error(t, err)
	})
	t.Run("Update", func(t *testing.T) {
		_, err := a.Update(ctx, "n-1", "title", "")
		require.Error(t, err)
	})
	t.Run("Delete", func(t *testing.T) {
		err := a.Delete(ctx, "n-1")
		require.Error(t, err)
	})
	t.Run("GetContent", func(t *testing.T) {
		_, err := a.GetContent(ctx, "n-1")
		require.Error(t, err)
	})
	t.Run("SetContent", func(t *testing.T) {
		err := a.SetContent(ctx, "n-1", "content")
		require.Error(t, err)
	})
	t.Run("Search", func(t *testing.T) {
		_, err := a.Search(ctx, "query")
		require.Error(t, err)
	})
	t.Run("GetAppInfo", func(t *testing.T) {
		_, err := a.GetAppInfo(ctx)
		require.Error(t, err)
	})
}
