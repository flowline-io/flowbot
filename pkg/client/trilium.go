package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/validate"
)

// TriliumClient provides access to the trilium note API.
type TriliumClient struct {
	c *Client
}

// ListNotesQuery contains query parameters for listing notes.
type ListNotesQuery struct {
	Limit  int
	Cursor string
	Query  string
}

// NoteListResult holds the paginated list response extracted from InvokeResult.
type NoteListResult struct {
	Items []*capability.Note `json:"data"`
	Page  NotePage           `json:"page"`
}

// NotePage holds pagination metadata.
type NotePage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// NoteItemResult holds a single note extracted from InvokeResult.
type NoteItemResult struct {
	Item capability.Note `json:"data"`
}

// NoteContentResult holds note content extracted from InvokeResult.
type NoteContentResult struct {
	Content string `json:"data"`
}

// List returns a paginated list of notes.
func (t *TriliumClient) List(ctx context.Context, query *ListNotesQuery) (*NoteListResult, error) {
	if query != nil {
		if err := validateListNotesQuery(query); err != nil {
			return nil, err
		}
	}
	path := "/service/trilium"
	if query != nil {
		v := url.Values{}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if query.Query != "" {
			v.Set("query", query.Query)
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}
	var result NoteListResult
	err := t.c.Get(ctx, path, &result)
	return &result, err
}

func validateListNotesQuery(query *ListNotesQuery) error {
	if query.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", query.Limit)
	}
	if query.Limit > validate.MaxSearchLimit {
		return fmt.Errorf("limit exceeds maximum of %d", validate.MaxSearchLimit)
	}
	if len(query.Cursor) > validate.QueryMaxLen {
		return fmt.Errorf("cursor exceeds maximum length of %d", validate.QueryMaxLen)
	}
	return nil
}

// Get returns a single note by ID.
func (t *TriliumClient) Get(ctx context.Context, id string) (*capability.Note, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	var result NoteItemResult
	path := fmt.Sprintf("/service/trilium/%s", url.PathEscape(id))
	err := t.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// CreateNoteRequest is the request body for creating a note.
type CreateNoteRequest struct {
	Title        string `json:"title"`
	Content      string `json:"content,omitempty"`
	Type         string `json:"type,omitempty"`
	ParentNoteID string `json:"parent_note_id,omitempty"`
}

// Create creates a new note.
func (t *TriliumClient) Create(ctx context.Context, req *CreateNoteRequest) (*capability.Note, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	var result NoteItemResult
	err := t.c.Post(ctx, "/service/trilium", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// UpdateNoteRequest is the request body for updating a note.
type UpdateNoteRequest struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

// Update updates an existing note.
func (t *TriliumClient) Update(ctx context.Context, id string, req *UpdateNoteRequest) (*capability.Note, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	var result NoteItemResult
	path := fmt.Sprintf("/service/trilium/%s", url.PathEscape(id))
	err := t.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// Delete removes a note by ID.
func (t *TriliumClient) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	path := fmt.Sprintf("/service/trilium/%s", url.PathEscape(id))
	return t.c.Delete(ctx, path, nil, nil)
}

// SearchNotesQuery contains query parameters for searching notes.
type SearchNotesQuery struct {
	Q string
}

// Search searches notes by query string.
func (t *TriliumClient) Search(ctx context.Context, query *SearchNotesQuery) (*NoteListResult, error) {
	if query == nil || query.Q == "" {
		return nil, fmt.Errorf("query is required")
	}
	path := "/service/trilium/search?" + url.Values{"q": {query.Q}}.Encode()
	var result NoteListResult
	err := t.c.Get(ctx, path, &result)
	return &result, err
}

// GetContent returns the full content of a note.
func (t *TriliumClient) GetContent(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("id is required")
	}
	var result NoteContentResult
	path := fmt.Sprintf("/service/trilium/%s/content", url.PathEscape(id))
	err := t.c.Get(ctx, path, &result)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// SetContent replaces the full content of a note.
func (t *TriliumClient) SetContent(ctx context.Context, id, content string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	path := fmt.Sprintf("/service/trilium/%s/content", url.PathEscape(id))
	return t.c.Put(ctx, path, content, nil)
}

// Health returns app info when the trilium backend is reachable.
func (t *TriliumClient) Health(ctx context.Context) (*capability.Note, error) {
	var result NoteItemResult
	err := t.c.Get(ctx, "/service/trilium/health", &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}
