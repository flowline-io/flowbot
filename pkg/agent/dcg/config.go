package dcg

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
)

//go:embed config.toml
var embeddedConfig []byte

var (
	materializeMu    sync.Mutex
	materializedPath string
)

// MaterializeConfig writes the embedded dcg.toml to a temp file once and returns its path.
func MaterializeConfig() (string, error) {
	materializeMu.Lock()
	defer materializeMu.Unlock()
	if materializedPath != "" {
		if _, err := os.Stat(materializedPath); err == nil {
			return materializedPath, nil
		}
		flog.Warn("[dcg] previous config temp missing path=%s; rewriting", materializedPath)
	}
	f, err := os.CreateTemp("", "flowbot-dcg-*.toml")
	if err != nil {
		return "", fmt.Errorf("dcg: create config temp: %w", err)
	}
	path := f.Name()
	if _, err := f.Write(embeddedConfig); err != nil {
		return "", errors.Join(
			fmt.Errorf("dcg: write config temp: %w", err),
			f.Close(),
			os.Remove(path),
		)
	}
	if err := f.Close(); err != nil {
		removeErr := os.Remove(path)
		if removeErr != nil {
			return "", errors.Join(fmt.Errorf("dcg: close config temp: %w", err), removeErr)
		}
		return "", fmt.Errorf("dcg: close config temp: %w", err)
	}
	materializedPath = path
	flog.Info("[dcg] materialized config path=%s bytes=%d", path, len(embeddedConfig))
	return path, nil
}
