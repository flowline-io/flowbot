package karakeep

import (
	"testing"

	"github.com/bytedance/sonic"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	t.Parallel()
	t.Run("karakeep constants", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "karakeep", ID)
		assert.Equal(t, "endpoint", EndpointKey)
		assert.Equal(t, "api_key", ApikeyKey)
		assert.Equal(t, 100, MaxPageSize)
	})
}

func TestNewKarakeep(t *testing.T) {
	t.Parallel()
	t.Run("constructor creates client", func(t *testing.T) {
		t.Parallel()
		client := NewKarakeep("https://api.karakeep.com", "test-api-key")
		assert.NotNil(t, client)
	})
}

func TestBookmarkUnmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal bookmark", func(t *testing.T) {
		t.Parallel()
		data := `{
			"id": "abc123",
			"createdAt": "2025-01-01T00:00:00Z",
			"modifiedAt": "2025-01-02T00:00:00Z",
			"title": "Example",
			"archived": true,
			"favourited": false,
			"taggingStatus": "success",
			"summarizationStatus": "success",
			"note": "note here",
			"summary": "a summary",
			"source": "api",
			"userId": "user-1",
			"tags": [
				{"id":"t1","name":"tag","attachedBy":"ai"}
			],
			"content": {
				"type": "link",
				"url": "https://example.com",
				"title": "Example page",
				"description": "desc",
				"imageUrl": "https://example.com/image.png"
			},
			"assets": [
				{"id":"a1","assetType":"linkHtmlContent","fileName":"file.html"}
			]
		}`

		var b Bookmark
		err := sonic.Unmarshal([]byte(data), &b)
		require.NoError(t, err)
		assert.Equal(t, "abc123", b.Id)
		require.NotNil(t, b.Source)
		assert.Equal(t, "api", *b.Source)
		require.Len(t, b.Assets, 1)
		assert.Equal(t, "file.html", b.Assets[0].FileName)
	})
}

func TestBookmark_GetTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		title    *string
		expected string
	}{
		{
			name:     "get title with value",
			title:    new("Hello World"),
			expected: "Hello World",
		},
		{
			name:     "get title nil",
			title:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := Bookmark{Title: tt.title}
			assert.Equal(t, tt.expected, b.GetTitle())
		})
	}
}

//go:fix inline
func strptr(s string) *string { return new(s) }

func TestBookmark_GetSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		summary  *string
		expected string
	}{
		{
			name:     "get summary with value",
			summary:  new("A great article"),
			expected: "A great article",
		},
		{
			name:     "get summary nil",
			summary:  nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := Bookmark{Summary: tt.summary}
			assert.Equal(t, tt.expected, b.GetSummary())
		})
	}
}

func TestBookmark_Fields(t *testing.T) {
	t.Parallel()
	t.Run("bookmark struct fields", func(t *testing.T) {
		t.Parallel()
		b := Bookmark{
			Id:         "bm1",
			CreatedAt:  "2025-01-01T00:00:00Z",
			Archived:   false,
			Favourited: true,
		}
		assert.Equal(t, "bm1", b.Id)
		assert.Equal(t, "2025-01-01T00:00:00Z", b.CreatedAt)
		assert.False(t, b.Archived)
		assert.True(t, b.Favourited)
	})
}

func TestBookmarkContent_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal bookmark content", func(t *testing.T) {
		t.Parallel()
		data := `{
			"type": "link",
			"url": "https://example.com",
			"title": "Test Page",
			"description": "A test page",
			"imageUrl": "https://example.com/img.png",
			"htmlContent": "<p>html</p>"
		}`

		var content BookmarkContent
		err := sonic.Unmarshal([]byte(data), &content)
		require.NoError(t, err)

		assert.Equal(t, "link", content.Type)
		assert.Equal(t, "https://example.com", content.Url)
		require.NotNil(t, content.Title)
		assert.Equal(t, "Test Page", *content.Title)
		require.NotNil(t, content.Description)
		assert.Equal(t, "A test page", *content.Description)
	})
}

func TestBookmarksResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal bookmarks response", func(t *testing.T) {
		t.Parallel()
		data := `{
			"bookmarks": [
				{"id": "b1", "createdAt": "2025-01-01T00:00:00Z", "archived": false, "favourited": false}
			],
			"nextCursor": "cursor123"
		}`

		var resp BookmarksResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Bookmarks, 1)
		assert.Equal(t, "b1", resp.Bookmarks[0].Id)
		assert.Equal(t, "cursor123", resp.NextCursor)
	})
}

func TestBookmarksResponse_Empty(t *testing.T) {
	t.Parallel()
	t.Run("empty bookmarks response", func(t *testing.T) {
		t.Parallel()
		data := `{"bookmarks": [], "nextCursor": ""}`
		var resp BookmarksResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Bookmarks)
		assert.Empty(t, resp.NextCursor)
	})
}

func TestTagsResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal tags response", func(t *testing.T) {
		t.Parallel()
		data := `{
			"tags": [
				{"id": "t1", "name": "go", "numBookmarks": 5.0}
			]
		}`

		var resp TagsResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Tags, 1)
		assert.Equal(t, "t1", resp.Tags[0].Id)
		assert.Equal(t, "go", resp.Tags[0].Name)
		assert.InEpsilon(t, float64(5.0), float64(resp.Tags[0].NumBookmarks), 0.001)
	})
}

func TestTagNumBookmarksByAttachedType(t *testing.T) {
	t.Parallel()
	t.Run("tag num bookmarks by attached type", func(t *testing.T) {
		t.Parallel()
		ai := float32(3.0)
		human := float32(2.0)
		nb := TagNumBookmarksByAttachedType{
			Ai:    &ai,
			Human: &human,
		}
		assert.InEpsilon(t, float64(3.0), float64(*nb.Ai), 0.001)
		assert.InEpsilon(t, float64(2.0), float64(*nb.Human), 0.001)
	})
}

func TestAttachTagsResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal attach tags response", func(t *testing.T) {
		t.Parallel()
		data := `{"attached": ["tag1", "tag2"]}`
		var resp AttachTagsResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		assert.Equal(t, []string{"tag1", "tag2"}, resp.Attached)
	})
}

func TestDetachTagsResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal detach tags response", func(t *testing.T) {
		t.Parallel()
		data := `{"detached": ["tag3"]}`
		var resp DetachTagsResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		assert.Equal(t, []string{"tag3"}, resp.Detached)
	})
}

func TestArchiveResponse_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal archive response", func(t *testing.T) {
		t.Parallel()
		data := `{"archived": true}`
		var resp ArchiveResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Archived)
	})
}

func TestBookmarkTagRequest(t *testing.T) {
	t.Parallel()
	t.Run("bookmark tag request", func(t *testing.T) {
		t.Parallel()
		req := BookmarkTagRequest{TagName: "test-tag"}
		assert.Equal(t, "test-tag", req.TagName)
	})
}

func TestBookmarksQuery_Defaults(t *testing.T) {
	t.Parallel()
	t.Run("query defaults", func(t *testing.T) {
		t.Parallel()
		q := BookmarksQuery{}
		assert.Equal(t, 0, q.Limit)
		assert.False(t, q.Archived)
		assert.False(t, q.Favourited)
		assert.Empty(t, q.Cursor)
	})
}

func TestBookmarksQuery_WithValues(t *testing.T) {
	t.Parallel()
	t.Run("query with values", func(t *testing.T) {
		t.Parallel()
		q := BookmarksQuery{
			Limit:      10,
			Archived:   true,
			Favourited: false,
			Cursor:     "abc",
		}
		assert.Equal(t, 10, q.Limit)
		assert.True(t, q.Archived)
		assert.False(t, q.Favourited)
		assert.Equal(t, "abc", q.Cursor)
	})
}

func TestSearchBookmarksQuery_Defaults(t *testing.T) {
	t.Parallel()
	t.Run("search query defaults", func(t *testing.T) {
		t.Parallel()
		q := SearchBookmarksQuery{}
		assert.Empty(t, q.Q)
		assert.Empty(t, q.SortOrder)
		assert.Equal(t, 0, q.Limit)
		assert.Empty(t, q.Cursor)
		assert.False(t, q.IncludeContent)
	})
}

func TestSearchBookmarksQuery_WithValues(t *testing.T) {
	t.Parallel()
	t.Run("search query with values", func(t *testing.T) {
		t.Parallel()
		q := SearchBookmarksQuery{
			Q:              "golang tutorials",
			SortOrder:      "relevance",
			Limit:          20,
			Cursor:         "cursor123",
			IncludeContent: true,
		}
		assert.Equal(t, "golang tutorials", q.Q)
		assert.Equal(t, "relevance", q.SortOrder)
		assert.Equal(t, 20, q.Limit)
		assert.Equal(t, "cursor123", q.Cursor)
		assert.True(t, q.IncludeContent)
	})
}

func TestCheckUrlResponse_Unmarshal_WithBookmarkId(t *testing.T) {
	t.Parallel()
	t.Run("check url response with bookmark id", func(t *testing.T) {
		t.Parallel()
		data := `{"bookmarkId": "abc123"}`
		var resp CheckUrlResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		require.NotNil(t, resp.BookmarkId)
		assert.Equal(t, "abc123", *resp.BookmarkId)
	})
}

func TestCheckUrlResponse_Unmarshal_NotFound(t *testing.T) {
	t.Parallel()
	t.Run("check url response not found", func(t *testing.T) {
		t.Parallel()
		data := `{"bookmarkId": null}`
		var resp CheckUrlResponse
		err := sonic.Unmarshal([]byte(data), &resp)
		require.NoError(t, err)
		assert.Nil(t, resp.BookmarkId)
	})
}

func TestBookmarkTagsInner_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal bookmark tags inner", func(t *testing.T) {
		t.Parallel()
		data := `{"id":"t1","name":"go","attachedBy":"ai"}`
		var tag BookmarkTagsInner
		err := sonic.Unmarshal([]byte(data), &tag)
		require.NoError(t, err)
		assert.Equal(t, "t1", tag.Id)
		assert.Equal(t, "go", tag.Name)
		assert.Equal(t, "ai", tag.AttachedBy)
	})
}

func TestBookmarksBookmarkIdAssets_Unmarshal(t *testing.T) {
	t.Parallel()
	t.Run("json unmarshal bookmark assets", func(t *testing.T) {
		t.Parallel()
		data := `{"id":"a1","assetType":"linkHtmlContent","fileName":"file.html"}`
		var asset BookmarksBookmarkIdAssets
		err := sonic.Unmarshal([]byte(data), &asset)
		require.NoError(t, err)
		assert.Equal(t, "a1", asset.Id)
		assert.Equal(t, "linkHtmlContent", asset.AssetType)
		assert.Equal(t, "file.html", asset.FileName)
	})
}
