package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flowline-io/flowbot/cmd/cli/internal/model"
)

const (
	kanbanFile       = "kanbans.json"
	kanbanFilePrefix = "kanbans."
)

// GetKanbanPath returns the path to the kanban file
func GetKanbanPath(profile string) (string, error) {
	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	filename := kanbanFile
	if profile != "" {
		filename = kanbanFilePrefix + profile + ".json"
	}
	return filepath.Join(cfgDir, filename), nil
}

// LoadKanbans loads all kanban boards from storage
func LoadKanbans(profile string) (*model.KanbanStore, error) {
	path, err := GetKanbanPath(profile)
	if err != nil {
		return nil, fmt.Errorf("get kanban path: %w", err)
	}

	store := &model.KanbanStore{Kanbans: []model.Kanban{}}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("read kanbans: %w", err)
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse kanbans: %w", err)
	}

	return store, nil
}

// SaveKanbans saves all kanban boards to storage
func SaveKanbans(store *model.KanbanStore, profile string) error {
	path, err := GetKanbanPath(profile)
	if err != nil {
		return fmt.Errorf("get kanban path: %w", err)
	}

	release, err := AcquireLock(path)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer release()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal kanbans: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save kanbans: %w", err)
	}

	return nil
}
