package scryfall

import (
	"context"
	"net/http"

	"github.com/go-resty/resty/v2"
)

const (
	ID = "scryfall"
)

type Scryfall struct{}

func NewScryfall() *Scryfall {
	return &Scryfall{}
}

func (s *Scryfall) CardsSearch(ctx context.Context, q string) ([]Card, error) {
	c := resty.New()
	resp, err := c.R().
		SetContext(ctx).
		SetQueryParam("q", q).
		SetResult(&SearchResult{}).
		Get("https://api.scryfall.com/cards/search")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*SearchResult)
		if result != nil {
			return result.Data, nil
		}
	}
	return nil, nil
}
