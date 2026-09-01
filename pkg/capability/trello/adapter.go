package trello

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	provider "github.com/flowline-io/flowbot/pkg/providers/trello"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-capability-trello-cursor-v1")

type client interface {
	ListBoards(ctx context.Context) ([]provider.Board, error)
	GetBoard(ctx context.Context, boardID string) (*provider.Board, error)
	ListLists(ctx context.Context, boardID string) ([]provider.List, error)
	ListCards(ctx context.Context, boardID string) ([]provider.Card, error)
	GetCard(ctx context.Context, cardID string) (*provider.Card, error)
	SearchCards(ctx context.Context, query string, limit int) ([]provider.Card, error)
	CreateCard(ctx context.Context, listID, name, desc string) (*provider.Card, error)
	UpdateCard(ctx context.Context, cardID, name, desc string) (*provider.Card, error)
	MoveCard(ctx context.Context, cardID, listID string) (*provider.Card, error)
	DeleteCard(ctx context.Context, cardID string) error
	RegisterWebhook(ctx context.Context, boardID, callbackURL, description string) (*provider.Webhook, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	GetMe(ctx context.Context) (*provider.Member, error)
}

// Adapter implements Service using the Trello provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

// New creates an Adapter from provider config. Returns nil when not configured.
func New() Service {
	if c := provider.GetClient(); c != nil {
		return NewWithClient(c)
	}
	return nil
}

// NewWithClient creates an Adapter with the given client.
func NewWithClient(c client) Service {
	return &Adapter{
		client:       c,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) checkClient() error {
	if a.client == nil {
		return types.Errorf(types.ErrUnavailable, "trello client not available")
	}
	return nil
}

func resolveBoardID(boardID string) (string, error) {
	if boardID != "" {
		return boardID, nil
	}
	if defaultID := provider.GetDefaultBoardID(); defaultID != "" {
		return defaultID, nil
	}
	return "", types.Errorf(types.ErrInvalidArgument, "board_id is required")
}

func (a *Adapter) ListBoards(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.TrelloBoard], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "trello list boards canceled", err)
	}
	boards, err := a.client.ListBoards(ctx)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello list boards failed", err)
	}
	pageReq := capability.PageRequest{}
	if q != nil {
		pageReq = q.Page
	}
	return capability.PaginateOffsetSlice(toBoards(boards), pageReq, a.cursorSecret, a.now(), string(hub.CapTrello)), nil
}

func (a *Adapter) GetBoard(ctx context.Context, boardID string) (*capability.TrelloBoard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveBoardID(boardID)
	if err != nil {
		return nil, err
	}
	board, err := a.client.GetBoard(ctx, resolved)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello get board failed", err)
	}
	return toBoard(board), nil
}

func (a *Adapter) ListLists(ctx context.Context, boardID string) ([]*capability.TrelloList, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveBoardID(boardID)
	if err != nil {
		return nil, err
	}
	lists, err := a.client.ListLists(ctx, resolved)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello list lists failed", err)
	}
	return toLists(lists), nil
}

func (a *Adapter) ListCards(ctx context.Context, boardID string, q *ListQuery) (*capability.ListResult[capability.TrelloCard], error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveBoardID(boardID)
	if err != nil {
		return nil, err
	}
	cards, err := a.client.ListCards(ctx, resolved)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello list cards failed", err)
	}
	pageReq := capability.PageRequest{}
	if q != nil {
		pageReq = q.Page
	}
	return capability.PaginateOffsetSlice(toCards(cards), pageReq, a.cursorSecret, a.now(), string(hub.CapTrello)), nil
}

func (a *Adapter) GetCard(ctx context.Context, cardID string) (*capability.TrelloCard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	if cardID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "card_id is required")
	}
	card, err := a.client.GetCard(ctx, cardID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello get card failed", err)
	}
	return toCard(card), nil
}

func (a *Adapter) SearchCards(ctx context.Context, query string, limit int) ([]*capability.TrelloCard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	cards, err := a.client.SearchCards(ctx, query, limit)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello search cards failed", err)
	}
	return toCards(cards), nil
}

func (a *Adapter) CreateCard(ctx context.Context, listID, name, desc string) (*capability.TrelloCard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	card, err := a.client.CreateCard(ctx, listID, name, desc)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello create card failed", err)
	}
	return toCard(card), nil
}

func (a *Adapter) UpdateCard(ctx context.Context, cardID, name, desc string) (*capability.TrelloCard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	card, err := a.client.UpdateCard(ctx, cardID, name, desc)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello update card failed", err)
	}
	return toCard(card), nil
}

func (a *Adapter) MoveCard(ctx context.Context, cardID, listID string) (*capability.TrelloCard, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	card, err := a.client.MoveCard(ctx, cardID, listID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello move card failed", err)
	}
	return toCard(card), nil
}

func (a *Adapter) DeleteCard(ctx context.Context, cardID string) error {
	if err := a.checkClient(); err != nil {
		return err
	}
	if err := a.client.DeleteCard(ctx, cardID); err != nil {
		return types.WrapError(types.ErrProvider, "trello delete card failed", err)
	}
	return nil
}

func (a *Adapter) RegisterWebhook(ctx context.Context, boardID, callbackURL, description string) (*capability.TrelloWebhook, error) {
	if err := a.checkClient(); err != nil {
		return nil, err
	}
	resolved, err := resolveBoardID(boardID)
	if err != nil {
		return nil, err
	}
	if callbackURL == "" {
		if provider.GetWebhookToken() == "" {
			return nil, types.Errorf(types.ErrInvalidArgument, "webhook_token must be configured to use default callback_url")
		}
		callbackURL = provider.DefaultWebhookCallbackURL()
	}
	webhook, err := a.client.RegisterWebhook(ctx, resolved, callbackURL, description)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "trello register webhook failed", err)
	}
	return toWebhook(webhook), nil
}

func (a *Adapter) DeleteWebhook(ctx context.Context, webhookID string) error {
	if err := a.checkClient(); err != nil {
		return err
	}
	if webhookID == "" {
		return types.Errorf(types.ErrInvalidArgument, "webhook_id is required")
	}
	if err := a.client.DeleteWebhook(ctx, webhookID); err != nil {
		return types.WrapError(types.ErrProvider, "trello delete webhook failed", err)
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := a.checkClient(); err != nil {
		return false, err
	}
	_, err := a.client.GetMe(ctx)
	if err != nil {
		flog.Warn("trello health check failed: %v", err)
		return false, nil
	}
	return true, nil
}

func cardEntityID(card *capability.TrelloCard) string {
	if card == nil {
		return ""
	}
	return card.ID
}

func eventRef(eventType, entityID string) capability.EventRef {
	return capability.EventRef{
		EventID:   types.Id(),
		EventType: eventType,
		EntityID:  entityID,
	}
}

func mutationResult(card *capability.TrelloCard, eventType, app, text string) *capability.InvokeResult {
	entityID := cardEntityID(card)
	ev := eventRef(eventType, entityID)
	return &capability.InvokeResult{
		Data: card,
		Text: text,
		Resource: &capability.ResourceMeta{
			EventID:  ev.EventID,
			EntityID: entityID,
			App:      app,
		},
		Events: []capability.EventRef{ev},
	}
}

func deleteCardResult(cardID, app string) *capability.InvokeResult {
	ev := eventRef(types.EventTrelloCardDeleted, cardID)
	return &capability.InvokeResult{
		Text: fmt.Sprintf("card deleted: %s", cardID),
		Resource: &capability.ResourceMeta{
			EventID:  ev.EventID,
			EntityID: cardID,
			App:      app,
		},
		Events: []capability.EventRef{ev},
	}
}
