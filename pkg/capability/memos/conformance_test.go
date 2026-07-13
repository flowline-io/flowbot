package memos

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/capability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/memos"
)

func newFakeClientFromConformanceConfig(cfg conformance.MemoConfig) *fakeClient {
	c := &fakeClient{
		getErr:            cfg.GetErr,
		listErr:           cfg.ListErr,
		createErr:         cfg.CreateErr,
		updateErr:         cfg.UpdateErr,
		deleteErr:         cfg.DeleteErr,
		getCurrentUserErr: cfg.HealthErr,
		listRawErr:        cfg.RawErr,
	}
	if cfg.GetItem != nil {
		c.getResp = toProviderMemo(cfg.GetItem)
	}
	if cfg.ListItems != nil {
		memos := make([]provider.Memo, len(cfg.ListItems))
		for i, item := range cfg.ListItems {
			memos[i] = *toProviderMemo(item)
		}
		c.listResp = &provider.ListMemosResponse{
			Memos:         memos,
			NextPageToken: cfg.ListNextCursor,
		}
	} else {
		c.listResp = &provider.ListMemosResponse{
			Memos: []provider.Memo{},
		}
	}
	if cfg.CreateItem != nil {
		c.createResp = toProviderMemo(cfg.CreateItem)
	}
	if cfg.UpdateItem != nil {
		c.updateResp = toProviderMemo(cfg.UpdateItem)
	}
	if cfg.HealthOk {
		c.getCurrentUser = &provider.User{Name: "users/1", Username: "admin"}
	} else if cfg.HealthErr == nil {
		c.getCurrentUserErr = errors.New("health check failed")
	}
	if cfg.RawItems != nil {
		items := make([]map[string]any, len(cfg.RawItems))
		for i, item := range cfg.RawItems {
			if m, ok := item.(map[string]any); ok {
				items[i] = m
			}
		}
		c.listRawItems = items
	}
	c.listRawCursor = cfg.RawCursor
	return c
}

func toProviderMemo(m *capability.Memo) *provider.Memo {
	if m == nil {
		return nil
	}
	p := &provider.Memo{
		Name:       m.Name,
		State:      m.State,
		Content:    m.Content,
		Visibility: m.Visibility,
		Tags:       m.Tags,
		Pinned:     m.Pinned,
		Creator:    m.Creator,
		Snippet:    m.Snippet,
	}
	if !m.CreateTime.IsZero() {
		p.CreateTime = &m.CreateTime
	}
	if !m.UpdateTime.IsZero() {
		p.UpdateTime = &m.UpdateTime
	}
	return p
}

func TestMemoConformance(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "runs memo conformance test suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conformance.RunMemoConformance(t, func(_ *testing.T, cfg conformance.MemoConfig) conformance.MemoService {
				c := newFakeClientFromConformanceConfig(cfg)
				return NewWithClient(c)
			})
		})
	}
}

func TestConformance_FakeClient_ImplementsClient(t *testing.T) {
	t.Run("fakeClient satisfies client interface", func(_ *testing.T) {
		var _ client = (*fakeClient)(nil)
	})
}

func TestToProviderMemo(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   *capability.Memo
		want *provider.Memo
	}{
		{
			name: "nil input returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "full memo conversion",
			in:   &capability.Memo{Name: "memos/1", Content: "Hello", Visibility: "PUBLIC", Pinned: true, Tags: []string{"tag1"}, CreateTime: now, UpdateTime: now},
			want: &provider.Memo{Name: "memos/1", Content: "Hello", Visibility: "PUBLIC", Pinned: true, Tags: []string{"tag1"}, CreateTime: &now, UpdateTime: &now},
		},
		{
			name: "minimal memo conversion",
			in:   &capability.Memo{Name: "memos/1", Content: "Hi"},
			want: &provider.Memo{Name: "memos/1", Content: "Hi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := toProviderMemo(tt.in)
			if tt.want == nil {
				require.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			require.Equal(t, tt.want.Name, result.Name)
			require.Equal(t, tt.want.Content, result.Content)
		})
	}
}
