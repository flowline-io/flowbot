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
// Existing rows are loaded once by name, new rows use CreateBulk, and updates run in one transaction.
// Duplicate names in the input keep the last occurrence.
func (s *HubStore) SaveHomelabApps(ctx context.Context, apps []homelab.App) error {
	if s == nil || s.client == nil || len(apps) == 0 {
		return nil
	}

	apps = collapseAppsByName(apps)
	now := time.Now()
	names := make([]string, len(apps))
	for i, homelabApp := range apps {
		names[i] = homelabApp.Name
	}
	existingRows, err := s.client.App.Query().Where(app.NameIn(names...)).All(ctx)
	if err != nil {
		return fmt.Errorf("hub: list apps: %w", err)
	}
	byName := firstAppByName(existingRows)

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("hub: begin save apps tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	creates, err := applyHomelabAppWrites(ctx, tx, apps, byName, now)
	if err != nil {
		return err
	}
	if len(creates) > 0 {
		if _, err := tx.App.CreateBulk(creates...).Save(ctx); err != nil {
			return fmt.Errorf("hub: create apps: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("hub: commit save apps: %w", err)
	}
	committed = true
	return nil
}

func collapseAppsByName(apps []homelab.App) []homelab.App {
	idx := make(map[string]int, len(apps))
	out := make([]homelab.App, 0, len(apps))
	for _, homelabApp := range apps {
		if i, ok := idx[homelabApp.Name]; ok {
			out[i] = homelabApp
			continue
		}
		idx[homelabApp.Name] = len(out)
		out = append(out, homelabApp)
	}
	return out
}

func firstAppByName(rows []*gen.App) map[string]*gen.App {
	byName := make(map[string]*gen.App, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		if _, ok := byName[row.Name]; !ok {
			byName[row.Name] = row
		}
	}
	return byName
}

func applyHomelabAppWrites(ctx context.Context, tx *gen.Tx, apps []homelab.App, byName map[string]*gen.App, now time.Time) ([]*gen.AppCreate, error) {
	creates := make([]*gen.AppCreate, 0)
	for _, homelabApp := range apps {
		info, err := appJSON(homelabApp)
		if err != nil {
			return nil, err
		}
		if existing, ok := byName[homelabApp.Name]; ok {
			if _, err := tx.App.UpdateOne(existing).
				SetPath(homelabApp.Path).
				SetStatus(string(homelabApp.Status)).
				SetDockerInfo(info).
				SetUpdatedAt(now).
				Save(ctx); err != nil {
				return nil, fmt.Errorf("hub: update app %s: %w", homelabApp.Name, err)
			}
			continue
		}
		creates = append(creates, tx.App.Create().
			SetName(homelabApp.Name).
			SetPath(homelabApp.Path).
			SetStatus(string(homelabApp.Status)).
			SetDockerInfo(info).
			SetCreatedAt(now).
			SetUpdatedAt(now))
	}
	return creates, nil
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
