// Package store provides database storage implementations.
package store

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pagedata"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ---------------------------------------------------------------------------
// PageDataStore
// ---------------------------------------------------------------------------

// PageDataStore persists shareable view page data keyed by opaque tokens.
type PageDataStore struct {
	client *gen.Client
}

// NewPageDataStore creates a PageDataStore with the given ent client.
func NewPageDataStore(client *gen.Client) *PageDataStore {
	return &PageDataStore{client: client}
}

// CreatePageData inserts a new page_data row.
func (s *PageDataStore) CreatePageData(ctx context.Context, token, pageType, title string, data types.KV, createdBy string, expiresAt *time.Time) error {
	if s == nil || s.client == nil {
		return nil
	}
	m := s.client.PageData.Create().
		SetToken(token).
		SetType(pageType).
		SetTitle(title).
		SetCreatedBy(createdBy)
	if len(data) > 0 {
		m.SetData(data)
	}
	if expiresAt != nil {
		m.SetExpiresAt(*expiresAt)
	}
	_, err := m.Save(ctx)
	return err
}

// GetPageDataByToken retrieves a page_data row by token. Returns nil if not found.
func (s *PageDataStore) GetPageDataByToken(ctx context.Context, token string) (*gen.PageData, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	pageData, err := s.client.PageData.Query().
		Where(pagedata.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pageData, nil
}

// DeletePageData removes a page_data row by token. Returns the number of deleted rows.
func (s *PageDataStore) DeletePageData(ctx context.Context, token string) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	affected, err := s.client.PageData.Delete().
		Where(pagedata.TokenEQ(token)).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// DeleteExpiredPageData removes rows where expires_at < now(). Returns the number of deleted rows.
func (s *PageDataStore) DeleteExpiredPageData(ctx context.Context) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	affected, err := s.client.PageData.Delete().
		Where(pagedata.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int64(affected), nil
}
