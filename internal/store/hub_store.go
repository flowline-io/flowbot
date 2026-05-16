package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/app"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// HubStore persists homelab discovery data to the database.
type HubStore struct {
	client *gen.Client
}

// NewHubStore returns a HubStore backed by the given Ent client.
func NewHubStore(client *gen.Client) *HubStore {
	return &HubStore{client: client}
}

// SaveHomelabApps upserts a batch of discovered homelab apps.
// Each app is looked up by name; existing rows are updated, new rows are created.
func (s *HubStore) SaveHomelabApps(apps []homelab.App) error {
	if s == nil || s.client == nil {
		return nil
	}
	if len(apps) == 0 {
		return nil
	}

	now := time.Now()
	ctx := context.Background()

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

func appJSON(ha homelab.App) (model.JSON, error) {
	raw, err := sonic.Marshal(ha)
	if err != nil {
		return nil, fmt.Errorf("marshal homelab app: %w", err)
	}
	var info model.JSON
	if err := info.Scan(raw); err != nil {
		return nil, fmt.Errorf("scan homelab app json: %w", err)
	}
	return info, nil
}
