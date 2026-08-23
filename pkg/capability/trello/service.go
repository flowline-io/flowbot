package trello

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination for list operations.
type ListQuery struct {
	Page capability.PageRequest
}

// Service defines the Trello capability contract.
type Service interface {
	ListBoards(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.TrelloBoard], error)
	GetBoard(ctx context.Context, boardID string) (*capability.TrelloBoard, error)
	ListLists(ctx context.Context, boardID string) ([]*capability.TrelloList, error)
	ListCards(ctx context.Context, boardID string, q *ListQuery) (*capability.ListResult[capability.TrelloCard], error)
	GetCard(ctx context.Context, cardID string) (*capability.TrelloCard, error)
	SearchCards(ctx context.Context, query string, limit int) ([]*capability.TrelloCard, error)
	CreateCard(ctx context.Context, listID, name, desc string) (*capability.TrelloCard, error)
	UpdateCard(ctx context.Context, cardID, name, desc string) (*capability.TrelloCard, error)
	MoveCard(ctx context.Context, cardID, listID string) (*capability.TrelloCard, error)
	DeleteCard(ctx context.Context, cardID string) error
	RegisterWebhook(ctx context.Context, boardID, callbackURL, description string) (*capability.TrelloWebhook, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	HealthCheck(ctx context.Context) (bool, error)
}
