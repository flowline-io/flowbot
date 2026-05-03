package conformance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	arc "github.com/flowline-io/flowbot/pkg/ability/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ArchiveConfig configures the fake backend for each archive conformance subtest.
type ArchiveConfig struct {
	AddItem  *ability.ArchiveItem
	AddErr   error
	GetItem  *ability.ArchiveItem
	GetErr   error
	SearchItems     []*ability.ArchiveItem
	SearchNextCursor string
	SearchErr       error
}

// ArchiveServiceFactory creates a fresh archive Service wired to a fake backend
// whose behavior is determined by the config parameter.
type ArchiveServiceFactory func(t *testing.T, cfg ArchiveConfig) arc.Service

// RunArchiveConformance runs the standard archive capability conformance suite.
func RunArchiveConformance(t *testing.T, factory ArchiveServiceFactory) {
	t.Run("add success", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{
			AddItem: &ability.ArchiveItem{ID: "snap-1", URL: "https://example.com", Status: "created"},
		})
		item, err := svc.Add(t.Context(), arc.AddRequest{URL: "https://example.com"})
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "https://example.com", item.URL)
	})

	t.Run("add timeout", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{})
		_, err := svc.Add(CanceledContext(), arc.AddRequest{URL: "https://example.com"})
		RequireTimeoutError(t, err)
	})

	t.Run("add empty url", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{})
		_, err := svc.Add(t.Context(), arc.AddRequest{})
		RequireInvalidArgError(t, err)
	})

	t.Run("add provider error", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{AddErr: assert.AnError})
		_, err := svc.Add(t.Context(), arc.AddRequest{URL: "https://example.com"})
		RequireProviderError(t, err)
	})

	t.Run("search timeout", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{})
		_, err := svc.Search(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("get timeout", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{})
		_, err := svc.Get(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("get empty id", func(t *testing.T) {
		svc := factory(t, ArchiveConfig{})
		_, err := svc.Get(t.Context(), "")
		RequireInvalidArgError(t, err)
	})
}
