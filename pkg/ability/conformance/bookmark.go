package conformance

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	bm "github.com/flowline-io/flowbot/pkg/ability/bookmark"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BookmarkConfig configures the fake backend for each bookmark conformance subtest.
// Fields set to non-nil/non-zero values become the fake responses; zero values
// produce empty/default success responses.
type BookmarkConfig struct {
	ListItems         []*ability.Bookmark
	ListNextCursor    string
	ListErr           error
	GetItem           *ability.Bookmark
	GetErr            error
	CreateItem        *ability.Bookmark
	CreateErr         error
	DeleteErr         error
	ArchiveResult     *bool
	ArchiveErr        error
	SearchItems       []*ability.Bookmark
	SearchNextCursor  string
	SearchErr         error
	AttachTagsErr     error
	DetachTagsErr     error
	CheckURLExists    bool
	CheckURLID        string
	CheckURLErr       error
}

// BookmarkServiceFactory creates a fresh bookmark Service wired to a fake backend
// whose behavior is determined by the config parameter.
type BookmarkServiceFactory func(t *testing.T, cfg BookmarkConfig) bm.Service

// RunBookmarkConformance runs the standard bookmark capability conformance suite.
// The factory must wire cfg into a fresh adapter and fake client for each subtest.
func RunBookmarkConformance(t *testing.T, factory BookmarkServiceFactory) {
	t.Run("list pagination", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{
			ListItems:      []*ability.Bookmark{{ID: "1", URL: "https://example.com", Title: "Example"}},
			ListNextCursor: "provider-next",
		})
		result, err := svc.List(t.Context(), &bm.ListQuery{Page: ability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		RequireListResult(t, result, 20, true)
		require.NotEmpty(t, result.Page.NextCursor)
		assert.Len(t, result.Items, 1)
	})

	t.Run("list empty", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		result, err := svc.List(t.Context(), &bm.ListQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Empty(t, result.Items)
		assert.Empty(t, result.Page.NextCursor)
	})

	t.Run("list nil query", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		result, err := svc.List(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.List(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{ListErr: assert.AnError})
		_, err := svc.List(t.Context(), &bm.ListQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get success", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{
			GetItem: &ability.Bookmark{ID: "1", URL: "https://example.com", Title: "Test"},
		})
		item, err := svc.Get(t.Context(), "1")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "1", item.ID)
	})

	t.Run("get timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Get(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("get empty id", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Get(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{GetErr: assert.AnError})
		_, err := svc.Get(t.Context(), "1")
		RequireProviderError(t, err)
	})

	t.Run("create success", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{
			CreateItem: &ability.Bookmark{ID: "new", URL: "https://new.example.com", Title: "New"},
		})
		item, err := svc.Create(t.Context(), "https://new.example.com")
		require.NoError(t, err)
		require.NotNil(t, item)
		assert.Equal(t, "https://new.example.com", item.URL)
	})

	t.Run("create timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Create(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("create empty url", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Create(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("create provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{CreateErr: assert.AnError})
		_, err := svc.Create(t.Context(), "https://example.com")
		RequireProviderError(t, err)
	})

	t.Run("delete timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.Delete(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("delete empty id", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.Delete(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("archive success", func(t *testing.T) {
		archived := true
		svc := factory(t, BookmarkConfig{ArchiveResult: &archived})
		result, err := svc.Archive(t.Context(), "1")
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("archive timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Archive(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("archive empty id", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Archive(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("archive provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{ArchiveErr: assert.AnError})
		_, err := svc.Archive(t.Context(), "1")
		RequireProviderError(t, err)
	})

	t.Run("search success", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{
			SearchItems: []*ability.Bookmark{{ID: "1", URL: "https://example.com", Title: "Example"}},
		})
		result, err := svc.Search(t.Context(), &bm.SearchQuery{Q: "test"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 1)
	})

	t.Run("search timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, err := svc.Search(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("search nil query", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		result, err := svc.Search(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("search provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{SearchErr: assert.AnError})
		_, err := svc.Search(t.Context(), &bm.SearchQuery{})
		RequireProviderError(t, err)
	})

	t.Run("attach tags success", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.AttachTags(t.Context(), "1", []string{"go"})
		require.NoError(t, err)
	})

	t.Run("attach tags timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.AttachTags(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("attach tags empty id", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.AttachTags(t.Context(), "", []string{"go"})
		RequireInvalidArgError(t, err)
	})

	t.Run("attach tags empty tags", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.AttachTags(t.Context(), "1", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("attach tags provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{AttachTagsErr: assert.AnError})
		err := svc.AttachTags(t.Context(), "1", []string{"go"})
		RequireProviderError(t, err)
	})

	t.Run("detach tags success", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.DetachTags(t.Context(), "1", []string{"go"})
		require.NoError(t, err)
	})

	t.Run("detach tags timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.DetachTags(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("detach tags empty id", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.DetachTags(t.Context(), "", []string{"go"})
		RequireInvalidArgError(t, err)
	})

	t.Run("detach tags empty tags", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		err := svc.DetachTags(t.Context(), "1", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("detach tags provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{DetachTagsErr: assert.AnError})
		err := svc.DetachTags(t.Context(), "1", []string{"go"})
		RequireProviderError(t, err)
	})

	t.Run("check url exists", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{CheckURLExists: true, CheckURLID: "bookmark-1"})
		exists, id, err := svc.CheckURL(t.Context(), "https://example.com")
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, "bookmark-1", id)
	})

	t.Run("check url timeout", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, _, err := svc.CheckURL(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("check url empty url", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{})
		_, _, err := svc.CheckURL(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("check url provider error", func(t *testing.T) {
		svc := factory(t, BookmarkConfig{CheckURLErr: assert.AnError})
		_, _, err := svc.CheckURL(t.Context(), "https://example.com")
		RequireProviderError(t, err)
	})
}
