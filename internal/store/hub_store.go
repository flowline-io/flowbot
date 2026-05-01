package store

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/homelab"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HubStore struct {
	db *gorm.DB
}

func NewHubStore(db *gorm.DB) *HubStore {
	return &HubStore{db: db}
}

func (s *HubStore) SaveHomelabApps(apps []homelab.App) error {
	if s == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	rows := make([]model.App, 0, len(apps))
	for _, app := range apps {
		info, err := appJSON(app)
		if err != nil {
			return err
		}
		rows = append(rows, model.App{
			Name:       app.Name,
			Path:       app.Path,
			Status:     model.AppStatus(app.Status),
			DockerInfo: info,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"path",
			"status",
			"docker_info",
			"updated_at",
		}),
	}).Create(&rows).Error
}

func appJSON(app homelab.App) (model.JSON, error) {
	raw, err := sonic.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("marshal homelab app: %w", err)
	}
	var info model.JSON
	if err := info.Scan(raw); err != nil {
		return nil, fmt.Errorf("scan homelab app json: %w", err)
	}
	return info, nil
}
