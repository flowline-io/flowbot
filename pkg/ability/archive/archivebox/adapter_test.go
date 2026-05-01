package archivebox

import (
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability/archive"
	provider "github.com/flowline-io/flowbot/pkg/providers/archivebox"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	data provider.Data
	resp *provider.Response
	err  error
}

func (f *fakeClient) Add(data provider.Data) (*provider.Response, error) {
	f.data = data
	return f.resp, f.err
}

func TestAddCreatesArchiveItem(t *testing.T) {
	client := &fakeClient{resp: &provider.Response{Success: true, Result: []string{"snapshot-id"}}}
	adapter := NewWithClient(client)

	item, err := adapter.Add(t.Context(), archive.AddRequest{URL: "https://example.com"})
	require.NoError(t, err)
	require.Equal(t, "snapshot-id", item.ID)
	require.Equal(t, "https://example.com", item.URL)
	require.Equal(t, []string{"https://example.com"}, client.data.Urls)
}

func TestAddWrapsProviderError(t *testing.T) {
	adapter := NewWithClient(&fakeClient{err: errors.New("boom")})

	_, err := adapter.Add(t.Context(), archive.AddRequest{URL: "https://example.com"})
	require.Error(t, err)
	require.True(t, errors.Is(err, types.ErrProvider))
}

func TestAddRejectsEmptyURL(t *testing.T) {
	adapter := NewWithClient(&fakeClient{})

	_, err := adapter.Add(t.Context(), archive.AddRequest{})
	require.Error(t, err)
	require.True(t, errors.Is(err, types.ErrInvalidArgument))
}
