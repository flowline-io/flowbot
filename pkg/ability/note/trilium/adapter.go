// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	notesvc "github.com/flowline-io/flowbot/pkg/ability/note"
	provider "github.com/flowline-io/flowbot/pkg/providers/trilium"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.Trilium methods used by this adapter.
type client interface {
	CreateNote(ctx context.Context, req provider.CreateNoteDef) (*provider.NoteWithBranch, error)
	GetNote(ctx context.Context, noteID string) (*provider.Note, error)
	PatchNote(ctx context.Context, noteID string, req provider.PatchNoteRequest) (*provider.Note, error)
	DeleteNote(ctx context.Context, noteID string) error
	SearchNotes(ctx context.Context, params provider.SearchParams) (*provider.SearchResponse, error)
	GetNoteContent(ctx context.Context, noteID string) (string, error)
	UpdateNoteContent(ctx context.Context, noteID, content string) error
	GetAppInfo(ctx context.Context) (*provider.AppInfo, error)
	ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error)
}

// Adapter implements note.Service using the Trilium provider client.
type Adapter struct {
	client client
}

// New creates an Adapter using the default provider client (reads config from YAML).
func New() notesvc.Service {
	return NewWithClient(provider.GetClient())
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) notesvc.Service {
	return &Adapter{client: c}
}

// List returns a paginated list of notes, optionally filtered by query.
// Trilium does not have a dedicated "list all" endpoint; this uses SearchNotes
// without a search string to return all notes, applying the query parameter as
// the search string when provided.
func (a *Adapter) List(ctx context.Context, q *notesvc.ListQuery) (*ability.ListResult[ability.Note], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if q == nil {
		q = &notesvc.ListQuery{}
	}
	limit := normalizedLimit(q.Page.Limit)
	params := provider.SearchParams{
		Search: q.Query,
		Limit:  limit,
	}
	resp, err := a.client.SearchNotes(ctx, params)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium list notes failed", err)
	}
	items := make([]*ability.Note, len(resp.Results))
	for i, n := range resp.Results {
		items[i] = toNote(&n)
	}
	return &ability.ListResult[ability.Note]{
		Items: items,
		Page: &ability.PageInfo{
			Limit:   limit,
			HasMore: len(resp.Results) >= limit,
		},
	}, nil
}

// Get returns a single note by its ID.
func (a *Adapter) Get(ctx context.Context, id string) (*ability.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	n, err := a.client.GetNote(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium get note failed", err)
	}
	return toNote(n), nil
}

// Create creates a new note with the given parameters.
// If typ is empty, defaults to "text".
func (a *Adapter) Create(ctx context.Context, title, content, typ, parentNoteID string) (*ability.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	if typ == "" {
		typ = "text"
	}
	req := provider.CreateNoteDef{
		ParentNoteID: parentNoteID,
		Title:        title,
		Type:         typ,
		Content:      content,
	}
	resp, err := a.client.CreateNote(ctx, req)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium create note failed", err)
	}
	return toNote(&resp.Note), nil
}

// Update updates a note's title and/or content.
// When content is non-empty, it calls UpdateNoteContent separately after the metadata patch.
func (a *Adapter) Update(ctx context.Context, id, title, content string) (*ability.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	// Patch metadata when title is provided.
	if title != "" {
		patchReq := provider.PatchNoteRequest{Title: title}
		_, err := a.client.PatchNote(ctx, id, patchReq)
		if err != nil {
			return nil, types.WrapError(types.ErrProvider, "trilium patch note failed", err)
		}
	}
	// Update content when provided.
	if content != "" {
		if err := a.client.UpdateNoteContent(ctx, id, content); err != nil {
			return nil, types.WrapError(types.ErrProvider, "trilium update note content failed", err)
		}
	}
	// Fetch the updated note to return fresh state.
	n, err := a.client.GetNote(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium get note after update failed", err)
	}
	return toNote(n), nil
}

// Delete removes a note by its ID.
func (a *Adapter) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := a.client.DeleteNote(ctx, id); err != nil {
		return types.WrapError(types.ErrProvider, "trilium delete note failed", err)
	}
	return nil
}

// GetContent retrieves the full content of a note.
func (a *Adapter) GetContent(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return "", types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	content, err := a.client.GetNoteContent(ctx, id)
	if err != nil {
		return "", types.WrapError(types.ErrProvider, "trilium get note content failed", err)
	}
	return content, nil
}

// SetContent sets the full content of a note.
func (a *Adapter) SetContent(ctx context.Context, id, content string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := a.client.UpdateNoteContent(ctx, id, content); err != nil {
		return types.WrapError(types.ErrProvider, "trilium set note content failed", err)
	}
	return nil
}

// Search searches notes by the given query string.
func (a *Adapter) Search(ctx context.Context, query string) (*ability.ListResult[ability.Note], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	params := provider.SearchParams{
		Search: query,
		Limit:  provider.MaxPageSize,
	}
	resp, err := a.client.SearchNotes(ctx, params)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium search notes failed", err)
	}
	items := make([]*ability.Note, len(resp.Results))
	for i, n := range resp.Results {
		items[i] = toNote(&n)
	}
	return &ability.ListResult[ability.Note]{Items: items}, nil
}

// GetAppInfo returns information about the running Trilium instance,
// mapped into a Note domain type for the ability layer.
func (a *Adapter) GetAppInfo(ctx context.Context) (*ability.Note, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	info, err := a.client.GetAppInfo(ctx)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trilium get app info failed", err)
	}
	return &ability.Note{
		ID:    info.InstanceName,
		Title: "Trilium Notes " + info.AppVersion,
		Type:  "app_info",
	}, nil
}

// ListRawEvents lists notes as raw events for polling support.
func (a *Adapter) ListRawEvents(ctx context.Context, cursor string) ([]any, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, next, err := a.client.ListRawEvents(ctx, cursor)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "trilium list raw events failed", err)
	}
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result, next, nil
}

// normalizedLimit clamps the provided limit to a valid range.
// Zero or negative values default to 50; values above MaxPageSize are capped.
func normalizedLimit(limit int) int {
	const defaultLimit = 50
	if limit <= 0 || limit > provider.MaxPageSize {
		return defaultLimit
	}
	return limit
}

// toNote maps a provider.Note to an ability.Note domain type.
func toNote(n *provider.Note) *ability.Note {
	if n == nil {
		return nil
	}
	return &ability.Note{
		ID:              n.NoteID,
		Title:           n.Title,
		Type:            n.Type,
		ParentNoteIDs:   n.ParentNoteIDs,
		ChildNoteIDs:    n.ChildNoteIDs,
		IsProtected:     n.IsProtected,
		DateCreated:     n.DateCreated,
		DateModified:    n.DateModified,
		UtcDateCreated:  n.UtcDateCreated,
		UtcDateModified: n.UtcDateModified,
	}
}

// Compile-time interface check.
var _ notesvc.Service = (*Adapter)(nil)
