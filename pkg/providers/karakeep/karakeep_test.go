package karakeep

import (
	"encoding/json"
	"testing"
)

func TestBookmarkUnmarshal(t *testing.T) {
	// example payload returned by GET /bookmarks/:bookmarkId
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
        "imageUrl": "https://example.com/image.png",
        "imageAssetId": "img-1",
        "screenshotAssetId": "ss-1",
        "pdfAssetId": "pdf-1",
        "fullPageArchiveAssetId": "fpa-1",
        "precrawledArchiveAssetId": "pca-1",
        "videoAssetId": "vid-1",
        "favicon": "fav.ico",
        "htmlContent": "<p>html</p>",
        "contentAssetId": "ca-1",
        "crawledAt": "2025-01-01T01:00:00Z",
        "crawlStatus": "success",
        "author": "auth",
        "publisher": "pub",
        "datePublished": "2024-12-31",
        "dateModified": "2025-01-01"
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
