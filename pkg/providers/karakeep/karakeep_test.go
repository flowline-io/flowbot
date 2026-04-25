package karakeep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	assert.Equal(t, "karakeep", ID)
	assert.Equal(t, "endpoint", EndpointKey)
	assert.Equal(t, "api_key", ApikeyKey)
	assert.Equal(t, 100, MaxPageSize)
}

func TestNewKarakeep(t *testing.T) {
	client := NewKarakeep("https://api.karakeep.com", "test-api-key")
	assert.NotNil(t, client)
}

func TestBookmarkUnmarshal(t *testing.T) {
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
	if err := json.Unmarshal([]byte(data), &b); err != nil {
		t.Fatalf("failed to unmarshal bookmark: %v", err)
	}

	if b.Id != "abc123" {
		t.Errorf("id mismatch: %s", b.Id)
	}
	if b.Source == nil || *b.Source != "api" {
		t.Errorf("source missing or wrong")
	}
	if len(b.Assets) != 1 || b.Assets[0].FileName != "file.html" {
		t.Errorf("asset not parsed")
	}
}

func TestBookmark_GetTitle(t *testing.T) {
	title := "Hello World"
	b := Bookmark{Title: &title}
	assert.Equal(t, "Hello World", b.GetTitle())

	b = Bookmark{Title: nil}
	assert.Equal(t, "", b.GetTitle())
}

func TestBookmark_GetSummary(t *testing.T) {
	summary := "A great article"
	b := Bookmark{Summary: &summary}
	assert.Equal(t, "A great article", b.GetSummary())

	b = Bookmark{Summary: nil}
	assert.Equal(t, "", b.GetSummary())
}

func TestBookmark_Fields(t *testing.T) {
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
}

func TestBookmarkContent_Unmarshal(t *testing.T) {
	data := `{
		"type": "link",
		"url": "https://example.com",
		"title": "Test Page",
		"description": "A test page",
		"imageUrl": "https://example.com/img.png",
		"htmlContent": "<p>html</p>"
	}`

	var content BookmarkContent
	err := json.Unmarshal([]byte(data), &content)
	require.NoError(t, err)

	assert.Equal(t, "link", content.Type)
	assert.Equal(t, "https://example.com", content.Url)
	require.NotNil(t, content.Title)
	assert.Equal(t, "Test Page", *content.Title)
	require.NotNil(t, content.Description)
	assert.Equal(t, "A test page", *content.Description)
}

func TestBookmarksResponse_Unmarshal(t *testing.T) {
	data := `{
		"bookmarks": [
			{"id": "b1", "createdAt": "2025-01-01T00:00:00Z", "archived": false, "favourited": false}
		],
		"nextCursor": "cursor123"
	}`

	var resp BookmarksResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)

	assert.Len(t, resp.Bookmarks, 1)
	assert.Equal(t, "b1", resp.Bookmarks[0].Id)
	assert.Equal(t, "cursor123", resp.NextCursor)
}

func TestBookmarksResponse_Empty(t *testing.T) {
	data := `{"bookmarks": [], "nextCursor": ""}`
	var resp BookmarksResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Bookmarks)
	assert.Empty(t, resp.NextCursor)
}

func TestTagsResponse_Unmarshal(t *testing.T) {
	data := `{
		"tags": [
			{"id": "t1", "name": "go", "numBookmarks": 5.0}
		]
	}`

	var resp TagsResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)

	assert.Len(t, resp.Tags, 1)
	assert.Equal(t, "t1", resp.Tags[0].Id)
	assert.Equal(t, "go", resp.Tags[0].Name)
	assert.Equal(t, float32(5.0), resp.Tags[0].NumBookmarks)
}

func TestTagNumBookmarksByAttachedType(t *testing.T) {
	ai := float32(3.0)
	human := float32(2.0)
	nb := TagNumBookmarksByAttachedType{
		Ai:    &ai,
		Human: &human,
	}
	assert.Equal(t, float32(3.0), *nb.Ai)
	assert.Equal(t, float32(2.0), *nb.Human)
}

func TestAttachTagsResponse_Unmarshal(t *testing.T) {
	data := `{"attached": ["tag1", "tag2"]}`
	var resp AttachTagsResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	assert.Equal(t, []string{"tag1", "tag2"}, resp.Attached)
}

func TestDetachTagsResponse_Unmarshal(t *testing.T) {
	data := `{"detached": ["tag3"]}`
	var resp DetachTagsResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	assert.Equal(t, []string{"tag3"}, resp.Detached)
}

func TestArchiveResponse_Unmarshal(t *testing.T) {
	data := `{"archived": true}`
	var resp ArchiveResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Archived)
}

func TestBookmarkTagRequest(t *testing.T) {
	req := BookmarkTagRequest{TagName: "test-tag"}
	assert.Equal(t, "test-tag", req.TagName)
}

func TestBookmarksQuery_Defaults(t *testing.T) {
	q := BookmarksQuery{}
	assert.Equal(t, 0, q.Limit)
	assert.False(t, q.Archived)
	assert.False(t, q.Favourited)
	assert.Empty(t, q.Cursor)
}

func TestBookmarksQuery_WithValues(t *testing.T) {
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
}

func TestSearchBookmarksQuery_Defaults(t *testing.T) {
	q := SearchBookmarksQuery{}
	assert.Empty(t, q.Q)
	assert.Empty(t, q.SortOrder)
	assert.Equal(t, 0, q.Limit)
	assert.Empty(t, q.Cursor)
	assert.False(t, q.IncludeContent)
}

func TestSearchBookmarksQuery_WithValues(t *testing.T) {
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
}

func TestCheckUrlResponse_Unmarshal_WithBookmarkId(t *testing.T) {
	data := `{"bookmarkId": "abc123"}`
	var resp CheckUrlResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.BookmarkId)
	assert.Equal(t, "abc123", *resp.BookmarkId)
}

func TestCheckUrlResponse_Unmarshal_NotFound(t *testing.T) {
	data := `{"bookmarkId": null}`
	var resp CheckUrlResponse
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp.BookmarkId)
}

func TestBookmarkTagsInner_Unmarshal(t *testing.T) {
	data := `{"id":"t1","name":"go","attachedBy":"ai"}`
	var tag BookmarkTagsInner
	err := json.Unmarshal([]byte(data), &tag)
	require.NoError(t, err)
	assert.Equal(t, "t1", tag.Id)
	assert.Equal(t, "go", tag.Name)
	assert.Equal(t, "ai", tag.AttachedBy)
}

func TestBookmarksBookmarkIdAssets_Unmarshal(t *testing.T) {
	data := `{"id":"a1","assetType":"linkHtmlContent","fileName":"file.html"}`
	var asset BookmarksBookmarkIdAssets
	err := json.Unmarshal([]byte(data), &asset)
	require.NoError(t, err)
	assert.Equal(t, "a1", asset.Id)
	assert.Equal(t, "linkHtmlContent", asset.AssetType)
	assert.Equal(t, "file.html", asset.FileName)
}
