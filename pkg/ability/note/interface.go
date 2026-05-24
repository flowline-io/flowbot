// Package note implements the note capability for note-taking systems.
package note

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// ListQuery wraps pagination for listing notes.
type ListQuery struct {
	Page  ability.PageRequest
	Query string // optional search query
}

// Service defines the note capability contract.
// Provider adapters implement this interface to bridge providers and invokers.
type Service interface {
	// List returns a paginated list of notes, optionally filtered by query.
	List(ctx context.Context, q *ListQuery) (*ability.ListResult[ability.Note], error)
	// Get returns a single note by its ID.
	Get(ctx context.Context, id string) (*ability.Note, error)
	// Create creates a new note with the given parameters.
	Create(ctx context.Context, title, content, typ, parentNoteID string) (*ability.Note, error)
	// Update updates a note's title and/or content.
	Update(ctx context.Context, id, title, content string) (*ability.Note, error)
	// Delete removes a note by its ID.
	Delete(ctx context.Context, id string) error
	// GetContent retrieves the full content of a note.
	GetContent(ctx context.Context, id string) (string, error)
	// SetContent sets the full content of a note.
	SetContent(ctx context.Context, id, content string) error
	// Search searches notes by the given query string.
	Search(ctx context.Context, query string) (*ability.ListResult[ability.Note], error)
	// GetAppInfo returns information about the running note server instance.
	GetAppInfo(ctx context.Context) (*ability.Note, error)
}
