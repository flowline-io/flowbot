package transmission

import (
	"context"
	"testing"

	"github.com/hekmon/transmissionrpc/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

type fakeClient struct {
	addResp    transmissionrpc.Torrent
	addErr     error
	listResp   []transmissionrpc.Torrent
	listErr    error
	stopErr    error
	removeErr  error
	lastAddURL string
	lastStop   []int64
	lastRemove []int64
}

func (f *fakeClient) TorrentAddUrl(_ context.Context, magnetUrl string) (transmissionrpc.Torrent, error) {
	f.lastAddURL = magnetUrl
	return f.addResp, f.addErr
}

func (f *fakeClient) TorrentGetAll(_ context.Context) ([]transmissionrpc.Torrent, error) {
	return f.listResp, f.listErr
}

func (f *fakeClient) TorrentStopIDs(_ context.Context, ids []int64) error {
	f.lastStop = append([]int64(nil), ids...)
	return f.stopErr
}

func (f *fakeClient) TorrentRemove(_ context.Context, ids []int64) error {
	f.lastRemove = append([]int64(nil), ids...)
	return f.removeErr
}

var _ client = (*fakeClient)(nil)

//go:fix inline
func ptrInt64(v int64) *int64 { return new(v) }

//go:fix inline
func ptrString(v string) *string { return new(v) }

//go:fix inline
func ptrStatus(v transmissionrpc.TorrentStatus) *transmissionrpc.TorrentStatus { return new(v) }

//go:fix inline
func ptrFloat64(v float64) *float64 { return new(v) }

func TestAdapter_AddTorrent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		client     *fakeClient
		input      AddTorrentInput
		wantID     int64
		wantErr    bool
		errContain string
	}{
		{
			name: "add success",
			client: &fakeClient{
				addResp: transmissionrpc.Torrent{
					ID:   new(int64(7)),
					Name: new("ubuntu.iso"),
				},
			},
			input:  AddTorrentInput{URL: "magnet:?xt=urn:btih:abc"},
			wantID: 7,
		},
		{
			name:       "missing url",
			client:     &fakeClient{},
			input:      AddTorrentInput{},
			wantErr:    true,
			errContain: "url is required",
		},
		{
			name:    "provider error",
			client:  &fakeClient{addErr: assert.AnError},
			input:   AddTorrentInput{URL: "magnet:?xt=urn:btih:abc"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClient(tt.client)
			got, err := svc.AddTorrent(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
			assert.Equal(t, tt.input.URL, tt.client.lastAddURL)
		})
	}
}

func TestAdapter_ListStopRemoveHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "list success",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{
					listResp: []transmissionrpc.Torrent{{
						ID:           new(int64(1)),
						Name:         new("a"),
						Status:       new(transmissionrpc.TorrentStatusDownload),
						PercentDone:  new(0.5),
						RateDownload: new(int64(1024)),
					}},
				})
				items, err := svc.ListTorrents(context.Background())
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, int64(1), items[0].ID)
				assert.Equal(t, "downloading", items[0].Status)
				assert.InDelta(t, 0.5, items[0].PercentDone, 0.0001)
			},
		},
		{
			name: "list provider error",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{listErr: assert.AnError})
				_, err := svc.ListTorrents(context.Background())
				require.Error(t, err)
			},
		},
		{
			name: "stop success",
			run: func(t *testing.T) {
				fc := &fakeClient{}
				svc := NewWithClient(fc)
				err := svc.StopTorrents(context.Background(), StopTorrentsInput{IDs: []int64{1, 2}})
				require.NoError(t, err)
				assert.Equal(t, []int64{1, 2}, fc.lastStop)
			},
		},
		{
			name: "stop missing ids",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{})
				err := svc.StopTorrents(context.Background(), StopTorrentsInput{})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "ids is required")
			},
		},
		{
			name: "remove success",
			run: func(t *testing.T) {
				fc := &fakeClient{}
				svc := NewWithClient(fc)
				err := svc.RemoveTorrents(context.Background(), RemoveTorrentsInput{IDs: []int64{9}})
				require.NoError(t, err)
				assert.Equal(t, []int64{9}, fc.lastRemove)
			},
		},
		{
			name: "health success",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{listResp: []transmissionrpc.Torrent{}})
				ok, err := svc.HealthCheck(context.Background())
				require.NoError(t, err)
				assert.True(t, ok)
			},
		},
		{
			name: "health unhealthy",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{listErr: assert.AnError})
				ok, err := svc.HealthCheck(context.Background())
				require.Error(t, err)
				assert.False(t, ok)
			},
		},
		{
			name: "canceled context",
			run: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				svc := NewWithClient(&fakeClient{})
				_, err := svc.ListTorrents(ctx)
				require.Error(t, err)
			},
		},
		{
			name: "toTorrent maps empty fields",
			run: func(t *testing.T) {
				got := toTorrent(transmissionrpc.Torrent{})
				require.NotNil(t, got)
				assert.Equal(t, int64(0), got.ID)
				assert.Empty(t, got.Name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestNew_NilWhenUnconfigured(t *testing.T) {
	t.Parallel()
	_ = New()
}

func TestAdapter_CompileTimeTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "service returns domain torrent",
			run: func(t *testing.T) {
				var item *capability.Torrent
				svc := NewWithClient(&fakeClient{
					addResp: transmissionrpc.Torrent{ID: new(int64(3)), Name: new("x")},
				})
				got, err := svc.AddTorrent(context.Background(), AddTorrentInput{URL: "magnet:?xt=urn:btih:x"})
				require.NoError(t, err)
				item = got
				assert.Equal(t, int64(3), item.ID)
			},
		},
		{
			name: "remove missing ids",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{})
				err := svc.RemoveTorrents(context.Background(), RemoveTorrentsInput{})
				require.Error(t, err)
			},
		},
		{
			name: "stop provider error",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{stopErr: assert.AnError})
				err := svc.StopTorrents(context.Background(), StopTorrentsInput{IDs: []int64{1}})
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
