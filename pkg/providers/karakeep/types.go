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
	ModifiedAt    *string `json:"modifiedAt,omitzero"`
	Title         *string `json:"title,omitzero"`
	Archived      bool    `json:"archived"`
	Favourited    bool    `json:"favourited"`
	TaggingStatus *string `json:"taggingStatus,omitzero"`
	// summarization status was added in API responses starting
	// with the new /bookmarks/:id endpoint and is not returned
	// by the older list call.  Use a pointer to distinguish
	// missing values.
	SummarizationStatus *string                     `json:"summarizationStatus,omitzero"`
	Source              *string                     `json:"source,omitzero"`
	UserId              *string                     `json:"userId,omitzero"`
	Note                *string                     `json:"note,omitzero"`
	Summary             *string                     `json:"summary,omitzero"`
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
	Title                    *string `json:"title,omitzero"`
	Description              *string `json:"description,omitzero"`
	ImageUrl                 *string `json:"imageUrl,omitzero"`
	ImageAssetId             *string `json:"imageAssetId,omitzero"`
	ScreenshotAssetId        *string `json:"screenshotAssetId,omitzero"`
	FullPageArchiveAssetId   *string `json:"fullPageArchiveAssetId,omitzero"`
	PrecrawledArchiveAssetId *string `json:"precrawledArchiveAssetId,omitzero"`
	VideoAssetId             *string `json:"videoAssetId,omitzero"`
	Favicon                  *string `json:"favicon,omitzero"`
	HtmlContent              *string `json:"htmlContent,omitzero"`
	CrawledAt                *string `json:"crawledAt,omitzero"`
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
	Ai    *float32 `json:"ai,omitzero"`
	Human *float32 `json:"human,omitzero"`
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
