package confluence

// Space is a Confluence space resource.
type Space struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// SpaceListResponse is the paginated space list response.
type SpaceListResponse struct {
	Results []Space `json:"results"`
	Start   int     `json:"start"`
	Limit   int     `json:"limit"`
	Size    int     `json:"size"`
}

// Page is a Confluence content page.
type Page struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Status  string        `json:"status,omitempty"`
	Title   string        `json:"title"`
	Space   *SpaceRef     `json:"space,omitempty"`
	Body    *Body         `json:"body,omitempty"`
	Version *Version      `json:"version,omitempty"`
	Links   *ContentLinks `json:"_links,omitempty"`
}

// SpaceRef is a minimal space reference on content.
type SpaceRef struct {
	Key  string `json:"key"`
	Name string `json:"name,omitempty"`
}

// Body holds page body representations.
type Body struct {
	Storage *StorageBody `json:"storage,omitempty"`
}

// StorageBody is Confluence storage-format content.
type StorageBody struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version is content version metadata.
type Version struct {
	Number int `json:"number"`
}

// ContentLinks holds API links for content.
type ContentLinks struct {
	WebUI string `json:"webui,omitempty"`
}

// PageListResponse is the paginated content list response.
type PageListResponse struct {
	Results []Page `json:"results"`
	Start   int    `json:"start"`
	Limit   int    `json:"limit"`
	Size    int    `json:"size"`
}

// SearchResponse is the CQL search response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Start   int            `json:"start"`
	Limit   int            `json:"limit"`
	Size    int            `json:"size"`
}

// SearchResult is one search hit.
type SearchResult struct {
	Content Page `json:"content"`
}

// CreatePageRequest is the API body for creating a page.
type CreatePageRequest struct {
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Space     map[string]string      `json:"space"`
	Body      map[string]StorageBody `json:"body"`
	Ancestors []map[string]string    `json:"ancestors,omitempty"`
}

// UpdatePageRequest is the API body for updating a page.
type UpdatePageRequest struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Body    map[string]StorageBody `json:"body,omitempty"`
	Version Version                `json:"version"`
}

// WebhookPayload is a normalized inbound automation webhook body.
type WebhookPayload struct {
	ID    string          `json:"id,omitempty"`
	Event string          `json:"event"`
	Page  *WebhookPageRef `json:"page"`
	Space *SpaceRef       `json:"space"`
}

// WebhookPageRef identifies a page in webhook payloads.
type WebhookPageRef struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}
