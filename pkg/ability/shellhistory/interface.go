package shellhistory

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type Entry struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Directory string    `json:"directory,omitempty"`
	Host      string    `json:"host,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchQuery struct {
	Page ability.PageRequest
	Q    string
	Host string
}

type Service interface {
	Search(ctx context.Context, q *SearchQuery) (*ability.ListResult[Entry], error)
	Recent(ctx context.Context, q *SearchQuery) (*ability.ListResult[Entry], error)
}
