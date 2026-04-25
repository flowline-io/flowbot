package karakeep

const (
	MaxPageSize = 100
)

type BookmarksResponse struct {
	Bookmarks  []Bookmark `json:"bookmarks"`
	NextCursor string     `json:"nextCursor"`
}

type TagsResponse struct {
	Tags []Tag `json:"tags"`
}

type AttachTagsResponse struct {
	Attached []string `json:"attached"`
}

type DetachTagsResponse struct {
	Detached []string `json:"detached"`
}

type ArchiveResponse struct {
	Archived bool `json:"archived"`
}

type CheckUrlResponse struct {
	BookmarkId *string `json:"bookmarkId"`
}

type BookmarkTagRequest struct {
	TagName string `json:"tag_name"`
}

type Bookmark struct {
	Id            string  `json:"id"`
	CreatedAt     string  `json:"createdAt"`
	ModifiedAt    *string `json:"modifiedAt"`
	Title         *string `json:"title,omitempty"`
	Archived      bool    `json:"archived"`
	Favourited    bool    `json:"favourited"`
	TaggingStatus *string `json:"taggingStatus,omitempty"`
	// summarization status was added in API responses starting
	// with the new /bookmarks/:id endpoint and is not returned
	// by the older list call.  Use a pointer to distinguish
	// missing values.
	SummarizationStatus *string                     `json:"summarizationStatus,omitempty"`
	Source              *string                     `json:"source,omitempty"`
	UserId              *string                     `json:"userId,omitempty"`
	Note                *string                     `json:"note,omitempty"`
	Summary             *string                     `json:"summary,omitempty"`
	Tags                []BookmarkTagsInner         `json:"tags"`
	Assets              []BookmarksBookmarkIdAssets `json:"assets"`
	Content             BookmarkContent             `json:"content"`
}

func (b Bookmark) GetTitle() string {
	if b.Title == nil {
		return ""
	}
	return *b.Title
}

func (b Bookmark) GetSummary() string {
	if b.Summary == nil {
		return ""
	}
	return *b.Summary
}

type BookmarkContent struct {
	Type                     string  `json:"type"`
	Url                      string  `json:"url"`
	Title                    *string `json:"title,omitempty"`
	Description              *string `json:"description,omitempty"`
	ImageUrl                 *string `json:"imageUrl,omitempty"`
	ImageAssetId             *string `json:"imageAssetId,omitempty"`
	ScreenshotAssetId        *string `json:"screenshotAssetId,omitempty"`
	FullPageArchiveAssetId   *string `json:"fullPageArchiveAssetId,omitempty"`
	PrecrawledArchiveAssetId *string `json:"precrawledArchiveAssetId,omitempty"`
	VideoAssetId             *string `json:"videoAssetId,omitempty"`
	Favicon                  *string `json:"favicon,omitempty"`
	HtmlContent              *string `json:"htmlContent,omitempty"`
	CrawledAt                *string `json:"crawledAt,omitempty"`
}

type BookmarkTagsInner struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	AttachedBy string `json:"attachedBy"`
}

type BookmarksBookmarkIdAssets struct {
	Id        string `json:"id"`
	AssetType string `json:"assetType"`
	FileName  string `json:"fileName"`
}

type Tag struct {
	Id                         string                        `json:"id"`
	Name                       string                        `json:"name"`
	NumBookmarks               float32                       `json:"numBookmarks"`
	NumBookmarksByAttachedType TagNumBookmarksByAttachedType `json:"numBookmarksByAttachedType"`
}

type TagNumBookmarksByAttachedType struct {
	Ai    *float32 `json:"ai,omitempty"`
	Human *float32 `json:"human,omitempty"`
}

type BookmarksQuery struct {
	Limit      int    `json:"limit"`
	Archived   bool   `json:"archived"`
	Favourited bool   `json:"favourited"`
	Cursor     string `json:"cursor"`
}

type SearchBookmarksQuery struct {
	Q              string `json:"q"`
	SortOrder      string `json:"sortOrder"`
	Limit          int    `json:"limit"`
	Cursor         string `json:"cursor"`
	IncludeContent bool   `json:"includeContent"`
}
