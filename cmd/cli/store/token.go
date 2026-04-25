// Package store provides local file storage for CLI data
package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDir  = ".config"
	appConfig  = "flowbot"
	tokenFile  = "token"
	profileSep = "."
)

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	cfgDir := filepath.Join(home, configDir, appConfig)
	if err := os.MkdirAll(cfgDir, 0750); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	return cfgDir, nil
}

// GetTokenPath returns the path to the token file
func GetTokenPath(profile string) (string, error) {
	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	filename := tokenFile
	if profile != "" {
		filename = tokenFile + profileSep + profile
	}
	return filepath.Join(cfgDir, filename), nil
}

// SaveToken saves the authentication token
func SaveToken(token, profile string) error {
	path, err := GetTokenPath(profile)
	if err != nil {
		return fmt.Errorf("get token path: %w", err)
	}
	if err := os.WriteFile(path, []byte(token), 0600); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

// LoadToken loads the authentication token
func LoadToken(profile string) (string, error) {
	path, err := GetTokenPath(profile)
	if err != nil {
		return "", fmt.Errorf("get token path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read token: %w", err)
	}
	return string(data), nil
}

// AcquireLock creates a lock file for concurrent access
func AcquireLock(lockPath string) (func(), error) {
	lockFile := lockPath + ".lock"
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("create lock file: %w", err)
	}

	_, err = fmt.Fprintf(f, "%d", os.Getpid())
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("write lock: %w", err)
	}

	return func() {
		_ = f.Close()
		_ = os.Remove(lockFile)
	}, nil
}
