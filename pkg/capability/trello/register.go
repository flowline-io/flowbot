// Package trello implements the Trello cloud capability.
package trello

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the trello capability with hub and invoker registry.
func Register(app string, svc Service) error {
	return capability.Register(buildSpec(app, svc))
}

// CatalogSpec returns capability metadata for documentation.
func CatalogSpec() capability.Spec {
	return buildSpec("", nil)
}

func buildSpec(app string, svc Service) capability.Spec {
	return capability.Spec{
		Type:        hub.CapTrello,
		App:         app,
		Description: "Trello cloud boards, lists, and cards",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: types.EventTrelloCardCreated, Description: "Fires when a card is created"},
			{Name: types.EventTrelloCardUpdated, Description: "Fires when a card is updated"},
			{Name: types.EventTrelloCardMoved, Description: "Fires when a card is moved"},
			{Name: types.EventTrelloCardDeleted, Description: "Fires when a card is deleted"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpListBoards, Description: "List boards", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input: []hub.ParamDef{
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
				},
				Handler: invokeListBoards(svc),
			},
			{
				Name: OpGetBoard, Description: "Get a board", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input:   []hub.ParamDef{{Name: "board_id", Type: "string", Required: false, Description: "Board ID"}},
				Handler: invokeGetBoard(svc),
			},
			{
				Name: OpListLists, Description: "List lists on a board", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input:   []hub.ParamDef{{Name: "board_id", Type: "string", Required: false, Description: "Board ID"}},
				Handler: invokeListLists(svc),
			},
			{
				Name: OpListCards, Description: "List cards on a board", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input: []hub.ParamDef{
					{Name: "board_id", Type: "string", Required: false, Description: "Board ID"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Pagination cursor"},
				},
				Handler: invokeListCards(svc),
			},
			{
				Name: OpGetCard, Description: "Get a card", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input:   []hub.ParamDef{{Name: "card_id", Type: "string", Required: true, Description: "Card ID"}},
				Handler: invokeGetCard(svc),
			},
			{
				Name: OpSearchCards, Description: "Search cards", Scopes: []string{auth.ScopeServiceTrelloRead},
				Input: []hub.ParamDef{
					{Name: "query", Type: "string", Required: true, Description: "Search query"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum results"},
				},
				Handler: invokeSearchCards(svc),
			},
			{
				Name: OpCreateCard, Description: "Create a card", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "list_id", Type: "string", Required: true, Description: "Target list ID"},
					{Name: "name", Type: "string", Required: true, Description: "Card title"},
					{Name: "desc", Type: "string", Required: false, Description: "Card description"},
				},
				Handler: invokeCreateCard(svc, app),
			},
			{
				Name: OpUpdateCard, Description: "Update a card", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "card_id", Type: "string", Required: true, Description: "Card ID"},
					{Name: "name", Type: "string", Required: false, Description: "New title"},
					{Name: "desc", Type: "string", Required: false, Description: "New description"},
				},
				Handler: invokeUpdateCard(svc, app),
			},
			{
				Name: OpMoveCard, Description: "Move a card to another list", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "card_id", Type: "string", Required: true, Description: "Card ID"},
					{Name: "list_id", Type: "string", Required: true, Description: "Target list ID"},
				},
				Handler: invokeMoveCard(svc, app),
			},
			{
				Name: OpDeleteCard, Description: "Delete a card", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "card_id", Type: "string", Required: true, Description: "Card ID"}},
				Handler: invokeDeleteCard(svc, app),
			},
			{
				Name: OpRegisterWebhook, Description: "Register a Trello board webhook", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "board_id", Type: "string", Required: false, Description: "Board ID"},
					{Name: "callback_url", Type: "string", Required: false, Description: "Callback URL"},
					{Name: "description", Type: "string", Required: false, Description: "Webhook description"},
				},
				Handler: invokeRegisterWebhook(svc),
			},
			{
				Name: OpDeleteWebhook, Description: "Delete a Trello webhook", Scopes: []string{auth.ScopeServiceTrelloWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "webhook_id", Type: "string", Required: true, Description: "Webhook ID"}},
				Handler: invokeDeleteWebhook(svc),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceTrelloRead},
				Handler: invokeHealth(svc),
			},
		},
	}
}

func invokeListBoards(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListBoards(ctx, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.TrelloBoard]{Items: []*capability.TrelloBoard{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetBoard(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		boardID, _ := capability.StringParam(params, "board_id")
		item, err := svc.GetBoard(ctx, boardID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.Name}, nil
	}
}

func invokeListLists(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		boardID, _ := capability.StringParam(params, "board_id")
		items, err := svc.ListLists(ctx, boardID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: items}, nil
	}
}

func invokeListCards(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		boardID, _ := capability.StringParam(params, "board_id")
		q := &ListQuery{Page: capability.PageRequestFromParams(params)}
		result, err := svc.ListCards(ctx, boardID, q)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[capability.TrelloCard]{Items: []*capability.TrelloCard{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGetCard(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		cardID, err := capability.RequiredString(params, "card_id")
		if err != nil {
			return nil, err
		}
		item, err := svc.GetCard(ctx, cardID)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: item.Name}, nil
	}
}

func invokeSearchCards(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		query, err := capability.RequiredString(params, "query")
		if err != nil {
			return nil, err
		}
		limit := 0
		if v, ok := capability.IntParam(params, "limit"); ok {
			limit = v
		}
		items, err := svc.SearchCards(ctx, query, limit)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: items}, nil
	}
}

func invokeCreateCard(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		listID, err := capability.RequiredString(params, "list_id")
		if err != nil {
			return nil, err
		}
		name, err := capability.RequiredString(params, "name")
		if err != nil {
			return nil, err
		}
		desc, _ := capability.StringParam(params, "desc")
		item, err := svc.CreateCard(ctx, listID, name, desc)
		if err != nil {
			return nil, err
		}
		return mutationResult(item, types.EventTrelloCardCreated, app, "card created: "+item.Name), nil
	}
}

func invokeUpdateCard(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		cardID, err := capability.RequiredString(params, "card_id")
		if err != nil {
			return nil, err
		}
		name, _ := capability.StringParam(params, "name")
		desc, _ := capability.StringParam(params, "desc")
		item, err := svc.UpdateCard(ctx, cardID, name, desc)
		if err != nil {
			return nil, err
		}
		return mutationResult(item, types.EventTrelloCardUpdated, app, "card updated"), nil
	}
}

func invokeMoveCard(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		cardID, err := capability.RequiredString(params, "card_id")
		if err != nil {
			return nil, err
		}
		listID, err := capability.RequiredString(params, "list_id")
		if err != nil {
			return nil, err
		}
		item, err := svc.MoveCard(ctx, cardID, listID)
		if err != nil {
			return nil, err
		}
		return mutationResult(item, types.EventTrelloCardMoved, app, "card moved"), nil
	}
}

func invokeDeleteCard(svc Service, app string) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		cardID, err := capability.RequiredString(params, "card_id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteCard(ctx, cardID); err != nil {
			return nil, err
		}
		return deleteCardResult(cardID, app), nil
	}
}

func invokeRegisterWebhook(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		boardID, _ := capability.StringParam(params, "board_id")
		callbackURL, _ := capability.StringParam(params, "callback_url")
		description, _ := capability.StringParam(params, "description")
		item, err := svc.RegisterWebhook(ctx, boardID, callbackURL, description)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: item, Text: "webhook registered"}, nil
	}
}

func invokeDeleteWebhook(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		webhookID, err := capability.RequiredString(params, "webhook_id")
		if err != nil {
			return nil, err
		}
		if err := svc.DeleteWebhook(ctx, webhookID); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Text: "webhook deleted"}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok}, nil
	}
}
