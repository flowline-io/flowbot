// Package trello implements the Trello cloud API provider.
package trello

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID                 = "trello"
	APIKeyKey          = "api_key"
	TokenKey           = "token"
	WebhookTokenKey    = "webhook_token"
	DefaultBoardIDKey  = "default_board_id"
	defaultAPIBaseURL  = "https://api.trello.com/1"
)

// Trello is an HTTP client for the Trello REST API.
type Trello struct {
	c      *resty.Client
	apiKey string
	token  string
}

// GetWebhookToken reads the inbound webhook query token from config.
func GetWebhookToken() string {
	tok, err := providers.GetConfig(ID, WebhookTokenKey)
	if err != nil {
		return ""
	}
	return tok.String()
}

// GetDefaultBoardID reads the optional default board id from config.
func GetDefaultBoardID() string {
	id, err := providers.GetConfig(ID, DefaultBoardIDKey)
	if err != nil {
		return ""
	}
	return id.String()
}

// GetClient builds a Trello client from vendors.trello config.
// Returns nil when api_key or token is not configured.
func GetClient() *Trello {
	apiKey, _ := providers.GetConfig(ID, APIKeyKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if apiKey.String() == "" || token.String() == "" {
		return nil
	}
	return NewTrello(apiKey.String(), token.String())
}

// NewTrello creates a Trello API client.
func NewTrello(apiKey, token string) *Trello {
	if apiKey == "" {
		return nil
	}
	c := utils.DefaultRestyClient()
	c.SetBaseURL(defaultAPIBaseURL)
	return &Trello{c: c, apiKey: apiKey, token: token}
}

func (t *Trello) req(ctx context.Context) *resty.Request {
	return t.c.R().
		SetContext(ctx).
		SetQueryParam("key", t.apiKey).
		SetQueryParam("token", t.token)
}

func checkStatus(resp *resty.Response, op string) error {
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return nil
	}
	if resp.StatusCode() == http.StatusNotFound {
		return types.Errorf(types.ErrNotFound, "trello %s: not found", op)
	}
	return fmt.Errorf("trello %s: unexpected status %d: %s", op, resp.StatusCode(), resp.String())
}

// ListBoards returns boards for the authenticated member.
func (t *Trello) ListBoards(ctx context.Context) ([]Board, error) {
	var boards []Board
	resp, err := t.req(ctx).SetResult(&boards).Get("/members/me/boards")
	if err != nil {
		return nil, fmt.Errorf("trello list boards: %w", err)
	}
	if err := checkStatus(resp, "list boards"); err != nil {
		return nil, err
	}
	return boards, nil
}

// GetBoard returns a board by id.
func (t *Trello) GetBoard(ctx context.Context, boardID string) (*Board, error) {
	if boardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "board_id is required")
	}
	board := &Board{}
	resp, err := t.req(ctx).
		SetPathParam("boardID", boardID).
		SetResult(board).
		Get("/boards/{boardID}")
	if err != nil {
		return nil, fmt.Errorf("trello get board: %w", err)
	}
	if err := checkStatus(resp, "get board"); err != nil {
		return nil, err
	}
	return board, nil
}

// ListLists returns lists on a board.
func (t *Trello) ListLists(ctx context.Context, boardID string) ([]List, error) {
	if boardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "board_id is required")
	}
	var lists []List
	resp, err := t.req(ctx).
		SetPathParam("boardID", boardID).
		SetResult(&lists).
		Get("/boards/{boardID}/lists")
	if err != nil {
		return nil, fmt.Errorf("trello list lists: %w", err)
	}
	if err := checkStatus(resp, "list lists"); err != nil {
		return nil, err
	}
	return lists, nil
}

// ListCards returns cards on a board.
func (t *Trello) ListCards(ctx context.Context, boardID string) ([]Card, error) {
	if boardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "board_id is required")
	}
	var cards []Card
	resp, err := t.req(ctx).
		SetPathParam("boardID", boardID).
		SetResult(&cards).
		Get("/boards/{boardID}/cards")
	if err != nil {
		return nil, fmt.Errorf("trello list cards: %w", err)
	}
	if err := checkStatus(resp, "list cards"); err != nil {
		return nil, err
	}
	return cards, nil
}

// GetCard returns a card by id.
func (t *Trello) GetCard(ctx context.Context, cardID string) (*Card, error) {
	if cardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "card_id is required")
	}
	card := &Card{}
	resp, err := t.req(ctx).
		SetPathParam("cardID", cardID).
		SetResult(card).
		Get("/cards/{cardID}")
	if err != nil {
		return nil, fmt.Errorf("trello get card: %w", err)
	}
	if err := checkStatus(resp, "get card"); err != nil {
		return nil, err
	}
	return card, nil
}

// SearchCards searches cards with the Trello search API.
func (t *Trello) SearchCards(ctx context.Context, query string, limit int) ([]Card, error) {
	if strings.TrimSpace(query) == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "query is required")
	}
	if limit <= 0 {
		limit = 20
	}
	result := &SearchResult{}
	resp, err := t.req(ctx).
		SetResult(result).
		SetQueryParam("query", query).
		SetQueryParam("modelTypes", "cards").
		SetQueryParam("cards_limit", fmt.Sprintf("%d", limit)).
		Get("/search")
	if err != nil {
		return nil, fmt.Errorf("trello search cards: %w", err)
	}
	if err := checkStatus(resp, "search cards"); err != nil {
		return nil, err
	}
	return result.Cards, nil
}

// CreateCard creates a card on a list.
func (t *Trello) CreateCard(ctx context.Context, listID, name, desc string) (*Card, error) {
	if listID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "list_id is required")
	}
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	card := &Card{}
	resp, err := t.req(ctx).
		SetResult(card).
		SetQueryParam("idList", listID).
		SetQueryParam("name", name).
		SetQueryParam("desc", desc).
		Post("/cards")
	if err != nil {
		return nil, fmt.Errorf("trello create card: %w", err)
	}
	if err := checkStatus(resp, "create card"); err != nil {
		return nil, err
	}
	return card, nil
}

// UpdateCard updates card fields.
func (t *Trello) UpdateCard(ctx context.Context, cardID, name, desc string) (*Card, error) {
	if cardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "card_id is required")
	}
	req := t.req(ctx).SetResult(&Card{})
	if name != "" {
		req.SetQueryParam("name", name)
	}
	if desc != "" {
		req.SetQueryParam("desc", desc)
	}
	resp, err := req.
		SetPathParam("cardID", cardID).
		Put("/cards/{cardID}")
	if err != nil {
		return nil, fmt.Errorf("trello update card: %w", err)
	}
	if err := checkStatus(resp, "update card"); err != nil {
		return nil, err
	}
	card, ok := resp.Result().(*Card)
	if !ok || card == nil {
		return nil, fmt.Errorf("trello update card: unexpected response type")
	}
	return card, nil
}

// MoveCard moves a card to another list.
func (t *Trello) MoveCard(ctx context.Context, cardID, listID string) (*Card, error) {
	if cardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "card_id is required")
	}
	if listID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "list_id is required")
	}
	card := &Card{}
	resp, err := t.req(ctx).
		SetPathParam("cardID", cardID).
		SetResult(card).
		SetQueryParam("idList", listID).
		Put("/cards/{cardID}")
	if err != nil {
		return nil, fmt.Errorf("trello move card: %w", err)
	}
	if err := checkStatus(resp, "move card"); err != nil {
		return nil, err
	}
	return card, nil
}

// DeleteCard deletes a card.
func (t *Trello) DeleteCard(ctx context.Context, cardID string) error {
	if cardID == "" {
		return types.Errorf(types.ErrInvalidArgument, "card_id is required")
	}
	resp, err := t.req(ctx).
		SetPathParam("cardID", cardID).
		Delete("/cards/{cardID}")
	if err != nil {
		return fmt.Errorf("trello delete card: %w", err)
	}
	return checkStatus(resp, "delete card")
}

// RegisterWebhook registers a board webhook with Trello.
func (t *Trello) RegisterWebhook(ctx context.Context, boardID, callbackURL, description string) (*Webhook, error) {
	if boardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "board_id is required")
	}
	if callbackURL == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "callback_url is required")
	}
	if description == "" {
		description = "flowbot trello webhook"
	}
	webhook := &Webhook{}
	resp, err := t.req(ctx).
		SetResult(webhook).
		SetQueryParam("idModel", boardID).
		SetQueryParam("callbackURL", callbackURL).
		SetQueryParam("description", description).
		Post("/webhooks")
	if err != nil {
		return nil, fmt.Errorf("trello register webhook: %w", err)
	}
	if err := checkStatus(resp, "register webhook"); err != nil {
		return nil, err
	}
	return webhook, nil
}

// DeleteWebhook deletes a webhook by id.
func (t *Trello) DeleteWebhook(ctx context.Context, webhookID string) error {
	if webhookID == "" {
		return types.Errorf(types.ErrInvalidArgument, "webhook_id is required")
	}
	resp, err := t.req(ctx).
		SetPathParam("webhookID", webhookID).
		Delete("/webhooks/{webhookID}")
	if err != nil {
		return fmt.Errorf("trello delete webhook: %w", err)
	}
	return checkStatus(resp, "delete webhook")
}

// GetMe returns the authenticated member (health check).
func (t *Trello) GetMe(ctx context.Context) (*Member, error) {
	member := &Member{}
	resp, err := t.req(ctx).SetResult(member).Get("/members/me")
	if err != nil {
		return nil, fmt.Errorf("trello get me: %w", err)
	}
	if err := checkStatus(resp, "get me"); err != nil {
		return nil, err
	}
	return member, nil
}

// DefaultWebhookCallbackURL builds the flowbot inbound webhook URL for Trello.
func DefaultWebhookCallbackURL() string {
	base := strings.TrimRight(types.AppUrl(), "/") + "/webhook/provider/trello/events"
	token := GetWebhookToken()
	if token == "" {
		return base
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}
