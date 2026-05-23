package example

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "github.com/flowline-io/flowbot/pkg/providers/example"
)

type fakeClient struct {
	getResp     *provider.Response
	getErr      error
	postResp    *provider.Response
	postErr     error
	putResp     *provider.Response
	putErr      error
	deleteResp  *provider.Response
	deleteErr   error
	statusResp  *provider.Response
	statusErr   error
	listRawResp []map[string]any
	listRawNext string
	listRawErr  error
}

func (f *fakeClient) Get(_ context.Context, _ string) (*provider.Response, error) {
	return f.getResp, f.getErr
}
func (f *fakeClient) Post(_ context.Context, _ string, _ any) (*provider.Response, error) {
	return f.postResp, f.postErr
}
func (f *fakeClient) Put(_ context.Context, _ string, _ any) (*provider.Response, error) {
	return f.putResp, f.putErr
}
func (f *fakeClient) Delete(_ context.Context, _ string) (*provider.Response, error) {
	return f.deleteResp, f.deleteErr
}
func (f *fakeClient) GetStatus(_ context.Context, _ int) (*provider.Response, error) {
	return f.statusResp, f.statusErr
}
func (f *fakeClient) ListRawEvents(_ context.Context, _ string) ([]map[string]any, string, error) {
	return f.listRawResp, f.listRawNext, f.listRawErr
}

func TestAdapter_GetItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{getResp: &provider.Response{Title: "hello", Body: "world"}},
			id:      "item-1",
			wantErr: false,
		},
		{
			name:    "empty id",
			client:  &fakeClient{},
			id:      "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: errors.New("down")},
			id:      "item-1",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.GetItem(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, item)
			assert.Equal(t, tt.id, item.ID)
		})
	}
}

func TestAdapter_ListItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{getResp: &provider.Response{Title: "hello"}},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "provider error",
			client:  &fakeClient{getErr: errors.New("timeout")},
			wantErr: true,
		},
		{
			name:    "success with nil response returns item",
			client:  &fakeClient{getResp: &provider.Response{}},
			wantLen: 1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListItems(context.Background(), nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestAdapter_CreateItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		title   string
		wantErr bool
	}{
		{name: "success", client: &fakeClient{postResp: &provider.Response{ID: 101}}, title: "test", wantErr: false},
		{name: "empty title", client: &fakeClient{}, title: "", wantErr: true},
		{name: "provider error", client: &fakeClient{postErr: errors.New("fail")}, title: "test", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			item, err := a.CreateItem(context.Background(), tt.title)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, item)
		})
	}
}

func TestAdapter_DeleteItem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		id      string
		wantErr bool
	}{
		{name: "success", client: &fakeClient{deleteResp: &provider.Response{}}, id: "item-1", wantErr: false},
		{name: "empty id", client: &fakeClient{}, id: "", wantErr: true},
		{name: "provider error", client: &fakeClient{deleteErr: errors.New("gone")}, id: "item-1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			err := a.DeleteItem(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAdapter_HealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantOk  bool
		wantErr bool
	}{
		{name: "healthy", client: &fakeClient{statusResp: &provider.Response{}}, wantOk: true, wantErr: false},
		{name: "unhealthy", client: &fakeClient{statusErr: errors.New("timeout")}, wantErr: true},
		{name: "success returns true", client: &fakeClient{statusResp: &provider.Response{Title: "hello"}}, wantOk: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			ok, err := a.HealthCheck(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestAdapter_ListRawEvents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		client     *fakeClient
		wantLen    int
		wantCursor string
		wantErr    bool
	}{
		{name: "success", client: &fakeClient{listRawResp: []map[string]any{{"id": "e1"}}}, wantLen: 1, wantCursor: "", wantErr: false},
		{name: "with cursor", client: &fakeClient{listRawResp: []map[string]any{{"id": "e1"}}, listRawNext: "next"}, wantLen: 1, wantCursor: "next", wantErr: false},
		{name: "provider error", client: &fakeClient{listRawErr: errors.New("down")}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			items, next, err := a.ListRawEvents(context.Background(), "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, items, tt.wantLen)
			assert.Equal(t, tt.wantCursor, next)
		})
	}
}

func TestAdapter_ContextCanceled(t *testing.T) {
	t.Run("canceled context returns timeout error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		a := NewWithClient(&fakeClient{getResp: &provider.Response{}})
		_, err := a.GetItem(ctx, "id")
		assert.Error(t, err)
	})
}
