// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/app"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// ---------------------------------------------------------------------------
// HubStore
// ---------------------------------------------------------------------------

// HubStore persists homelab discovery data to the database.
type HubStore struct {
	client *gen.Client
}

// AppInfo is a lightweight projection of store-level app metadata.
type AppInfo struct {
	Name      string
	UpdatedAt time.Time
}

// ListApps returns all apps from the database with Name and UpdatedAt.
// When the client is nil, returns nil (safe for no-DB environments).
func (s *HubStore) ListApps(ctx context.Context) ([]AppInfo, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rows, err := s.client.App.Query().Select(app.FieldName, app.FieldUpdatedAt).Order(app.ByName()).All(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]AppInfo, len(rows))
	for i, r := range rows {
		infos[i] = AppInfo{Name: r.Name, UpdatedAt: r.UpdatedAt}
	}
	return infos, nil
}

// NewHubStore returns a HubStore backed by the given Ent client.
func NewHubStore(client *gen.Client) *HubStore {
	return &HubStore{client: client}
}

// SaveHomelabApps upserts a batch of discovered homelab apps.
// Each app is looked up by name; existing rows are updated, new rows are created.
func (s *HubStore) SaveHomelabApps(ctx context.Context, apps []homelab.App) error {
	if s == nil || s.client == nil {
		return nil
	}
	if len(apps) == 0 {
		return nil
	}

	now := time.Now()

	for _, homelabApp := range apps {
		info, err := appJSON(homelabApp)
		if err != nil {
			return err
		}

		existing, err := s.client.App.Query().
			Where(app.NameEQ(homelabApp.Name)).
			First(ctx)
		if err != nil {
			if !gen.IsNotFound(err) {
				return err
			}
			// Not found: create.
			_, createErr := s.client.App.Create().
				SetName(homelabApp.Name).
				SetPath(homelabApp.Path).
				SetStatus(string(homelabApp.Status)).
				SetDockerInfo(info).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			if createErr != nil {
				return createErr
			}
		} else {
			// Found: update.
			_, updateErr := s.client.App.UpdateOne(existing).
				SetPath(homelabApp.Path).
				SetStatus(string(homelabApp.Status)).
				SetDockerInfo(info).
				SetUpdatedAt(now).
				Save(ctx)
			if updateErr != nil {
				return updateErr
			}
		}
	}

	return nil
}

func appJSON(ha homelab.App) (schema.JSON, error) {
	raw, err := sonic.Marshal(ha)
	if err != nil {
		return nil, fmt.Errorf("marshal homelab app: %w", err)
	}
	var info schema.JSON
	if err := info.Scan(raw); err != nil {
		return nil, fmt.Errorf("scan homelab app json: %w", err)
	}
	return info, nil
}
