package trello

import (
	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/trello"
)

func toBoard(b *provider.Board) *capability.TrelloBoard {
	if b == nil {
		return nil
	}
	return &capability.TrelloBoard{
		ID:     b.ID,
		Name:   b.Name,
		Desc:   b.Desc,
		Closed: b.Closed,
		URL:    b.URL,
	}
}

func toBoards(items []provider.Board) []*capability.TrelloBoard {
	out := make([]*capability.TrelloBoard, len(items))
	for i := range items {
		out[i] = toBoard(&items[i])
	}
	return out
}

func toList(l *provider.List) *capability.TrelloList {
	if l == nil {
		return nil
	}
	return &capability.TrelloList{
		ID:      l.ID,
		Name:    l.Name,
		Closed:  l.Closed,
		IDBoard: l.IDBoard,
		Pos:     l.Pos,
	}
}

func toLists(items []provider.List) []*capability.TrelloList {
	out := make([]*capability.TrelloList, len(items))
	for i := range items {
		out[i] = toList(&items[i])
	}
	return out
}

func toCard(c *provider.Card) *capability.TrelloCard {
	if c == nil {
		return nil
	}
	return &capability.TrelloCard{
		ID:       c.ID,
		Name:     c.Name,
		Desc:     c.Desc,
		IDBoard:  c.IDBoard,
		IDList:   c.IDList,
		Pos:      c.Pos,
		Closed:   c.Closed,
		URL:      c.URL,
		ShortURL: c.ShortURL,
	}
}

func toCards(items []provider.Card) []*capability.TrelloCard {
	out := make([]*capability.TrelloCard, len(items))
	for i := range items {
		out[i] = toCard(&items[i])
	}
	return out
}

func toWebhook(w *provider.Webhook) *capability.TrelloWebhook {
	if w == nil {
		return nil
	}
	return &capability.TrelloWebhook{
		ID:          w.ID,
		Description: w.Description,
		IDModel:     w.IDModel,
		CallbackURL: w.CallbackURL,
		Active:      w.Active,
	}
}
