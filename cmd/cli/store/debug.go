package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const debugFileName = "debug"

// GetDebugPath returns the path to the debug config file.
func GetDebugPath(profile string) (string, error) {
	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	filename := debugFileName
	if profile != "" {
		filename = debugFileName + profileSep + profile
	}
	return filepath.Join(cfgDir, filename), nil
}

// SaveDebug saves the debug mode setting.
func SaveDebug(enabled bool, profile string) error {
	path, err := GetDebugPath(profile)
	if err != nil {
		return fmt.Errorf("get debug path: %w", err)
	}
	val := "false"
	if enabled {
		val = "true"
	}
	if err := os.WriteFile(path, []byte(val), 0600); err != nil {
		return fmt.Errorf("save debug: %w", err)
	}
	return nil
}

// LoadDebug loads the debug mode setting.
// Returns false if the setting has never been saved.
func LoadDebug(profile string) (bool, error) {
	path, err := GetDebugPath(profile)
	if err != nil {
		return false, fmt.Errorf("get debug path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read debug: %w", err)
	}
	return string(data) == "true", nil
}
