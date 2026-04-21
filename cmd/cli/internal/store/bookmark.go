package store

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/internal/model"
)

const (
	bookmarkFile       = "bookmarks.json"
	bookmarkFilePrefix = "bookmarks."
	maxTitleLength     = 200
	maxDescLength      = 1000
	maxURLLength       = 2048
)

// ValidateBookmark validates bookmark fields
func ValidateBookmark(title, urlStr, description string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(title) > maxTitleLength {
		return fmt.Errorf("title too long (max %d characters)", maxTitleLength)
	}
	if len(urlStr) > maxURLLength {
		return fmt.Errorf("URL too long (max %d characters)", maxURLLength)
	}
	if len(description) > maxDescLength {
		return fmt.Errorf("description too long (max %d characters)", maxDescLength)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// GetBookmarkPath returns the path to the bookmark file
func GetBookmarkPath(profile string) (string, error) {
	cfgDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	filename := bookmarkFile
	if profile != "" {
		filename = bookmarkFilePrefix + profile + ".json"
	}
	return filepath.Join(cfgDir, filename), nil
}

// LoadBookmarks loads all bookmarks from storage
func LoadBookmarks(profile string) (*model.BookmarkStore, error) {
	path, err := GetBookmarkPath(profile)
	if err != nil {
		return nil, fmt.Errorf("get bookmark path: %w", err)
	}

	store := &model.BookmarkStore{Bookmarks: []model.Bookmark{}}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("read bookmarks: %w", err)
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("parse bookmarks: %w", err)
	}

	return store, nil
}

// SaveBookmarks saves all bookmarks to storage
func SaveBookmarks(store *model.BookmarkStore, profile string) error {
	path, err := GetBookmarkPath(profile)
	if err != nil {
		return fmt.Errorf("get bookmark path: %w", err)
	}

	release, err := AcquireLock(path)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer release()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bookmarks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save bookmarks: %w", err)
	}

	return nil
}
