package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// TrelloClient provides access to the Trello API via Flowbot server.
type TrelloClient struct {
	c *Client
}

// TrelloListResult holds a paginated list response from InvokeResult.
type TrelloListResult[T any] struct {
	Data []T        `json:"data"`
	Page TrelloPage `json:"page"`
}

// TrelloPage holds pagination metadata.
type TrelloPage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// TrelloItemResult holds a single item from InvokeResult.
type TrelloItemResult[T any] struct {
	Data T `json:"data"`
}

// ListBoards returns boards for the authenticated member.
func (t *TrelloClient) ListBoards(ctx context.Context, limit int, cursor string) (*TrelloListResult[capability.TrelloBoard], error) {
	path := trelloPathWithPage("/service/trello/boards", limit, cursor)
	var result TrelloListResult[capability.TrelloBoard]
	err := t.c.Get(ctx, path, &result)
	return &result, err
}

// GetBoard returns a board by id.
func (t *TrelloClient) GetBoard(ctx context.Context, boardID string) (*capability.TrelloBoard, error) {
	if boardID == "" {
		return nil, fmt.Errorf("board_id is required")
	}
	var result TrelloItemResult[capability.TrelloBoard]
	path := fmt.Sprintf("/service/trello/boards/%s", url.PathEscape(boardID))
	err := t.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// ListLists returns lists on a board.
func (t *TrelloClient) ListLists(ctx context.Context, boardID string) ([]capability.TrelloList, error) {
	if boardID == "" {
		return nil, fmt.Errorf("board_id is required")
	}
	var result []capability.TrelloList
	path := fmt.Sprintf("/service/trello/boards/%s/lists", url.PathEscape(boardID))
	err := t.c.Get(ctx, path, &result)
	return result, err
}

// ListCards returns cards on a board.
func (t *TrelloClient) ListCards(ctx context.Context, boardID string, limit int, cursor string) (*TrelloListResult[capability.TrelloCard], error) {
	if boardID == "" {
		return nil, fmt.Errorf("board_id is required")
	}
	base := fmt.Sprintf("/service/trello/boards/%s/cards", url.PathEscape(boardID))
	path := trelloPathWithPage(base, limit, cursor)
	var result TrelloListResult[capability.TrelloCard]
	err := t.c.Get(ctx, path, &result)
	return &result, err
}

// GetCard returns a card by id.
func (t *TrelloClient) GetCard(ctx context.Context, cardID string) (*capability.TrelloCard, error) {
	if cardID == "" {
		return nil, fmt.Errorf("card_id is required")
	}
	var result TrelloItemResult[capability.TrelloCard]
	path := fmt.Sprintf("/service/trello/cards/%s", url.PathEscape(cardID))
	err := t.c.Get(ctx, path, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// SearchCards searches cards by query.
func (t *TrelloClient) SearchCards(ctx context.Context, query string, limit int) ([]capability.TrelloCard, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	v := url.Values{"q": {query}}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	var result []capability.TrelloCard
	path := "/service/trello/search?" + v.Encode()
	err := t.c.Get(ctx, path, &result)
	return result, err
}

// CreateCardRequest is the body for creating a card.
type CreateCardRequest struct {
	ListID string `json:"list_id"`
	Name   string `json:"name"`
	Desc   string `json:"desc,omitempty"`
}

// CreateCard creates a new card.
func (t *TrelloClient) CreateCard(ctx context.Context, req *CreateCardRequest) (*capability.TrelloCard, error) {
	if req == nil || req.ListID == "" || req.Name == "" {
		return nil, fmt.Errorf("list_id and name are required")
	}
	var result TrelloItemResult[capability.TrelloCard]
	err := t.c.Post(ctx, "/service/trello/cards", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateCardRequest is the body for updating a card.
type UpdateCardRequest struct {
	Name string `json:"name,omitempty"`
	Desc string `json:"desc,omitempty"`
}

// UpdateCard updates a card.
func (t *TrelloClient) UpdateCard(ctx context.Context, cardID string, req *UpdateCardRequest) (*capability.TrelloCard, error) {
	if cardID == "" {
		return nil, fmt.Errorf("card_id is required")
	}
	if req == nil {
		req = &UpdateCardRequest{}
	}
	var result TrelloItemResult[capability.TrelloCard]
	path := fmt.Sprintf("/service/trello/cards/%s", url.PathEscape(cardID))
	err := t.c.Patch(ctx, path, req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// MoveCardRequest is the body for moving a card.
type MoveCardRequest struct {
	ListID string `json:"list_id"`
}

// MoveCard moves a card to another list.
func (t *TrelloClient) MoveCard(ctx context.Context, cardID, listID string) (*capability.TrelloCard, error) {
	if cardID == "" || listID == "" {
		return nil, fmt.Errorf("card_id and list_id are required")
	}
	var result TrelloItemResult[capability.TrelloCard]
	path := fmt.Sprintf("/service/trello/cards/%s/move", url.PathEscape(cardID))
	err := t.c.Post(ctx, path, &MoveCardRequest{ListID: listID}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteCard deletes a card.
func (t *TrelloClient) DeleteCard(ctx context.Context, cardID string) error {
	if cardID == "" {
		return fmt.Errorf("card_id is required")
	}
	path := fmt.Sprintf("/service/trello/cards/%s", url.PathEscape(cardID))
	return t.c.Delete(ctx, path, nil, nil)
}

// RegisterWebhookRequest registers a Trello webhook.
type RegisterWebhookRequest struct {
	BoardID     string `json:"board_id,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
	Description string `json:"description,omitempty"`
}

// RegisterWebhook registers a board webhook with Trello.
func (t *TrelloClient) RegisterWebhook(ctx context.Context, req *RegisterWebhookRequest) (*capability.TrelloWebhook, error) {
	if req == nil {
		req = &RegisterWebhookRequest{}
	}
	var result TrelloItemResult[capability.TrelloWebhook]
	err := t.c.Post(ctx, "/service/trello/webhooks", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteWebhook deletes a webhook by id.
func (t *TrelloClient) DeleteWebhook(ctx context.Context, webhookID string) error {
	if webhookID == "" {
		return fmt.Errorf("webhook_id is required")
	}
	path := fmt.Sprintf("/service/trello/webhooks/%s", url.PathEscape(webhookID))
	return t.c.Delete(ctx, path, nil, nil)
}

// Health checks Trello connectivity.
func (t *TrelloClient) Health(ctx context.Context) (bool, error) {
	var result bool
	err := t.c.Get(ctx, "/service/trello/health", &result)
	return result, err
}

func trelloPathWithPage(base string, limit int, cursor string) string {
	v := url.Values{}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}
