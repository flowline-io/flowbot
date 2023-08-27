package channels

import (
	"errors"
	"github.com/sysatom/flowbot/pkg/channels/crawler"
	"gopkg.in/yaml.v2"
	"io/fs"
	"os"
	"path/filepath"
)

const ChannelNameSuffix = "_channel"

type Publisher *crawler.Rule

var publishers map[string]Publisher

// Init initializes registered publishers.
func Init() error {
	configPath := os.Getenv("CHANNEL_PATH")
	if configPath == "" {
		return errors.New("channel failed to parse config env")
	}

	if publishers == nil {
		publishers = make(map[string]Publisher)
	}

	return filepath.Walk(configPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ext := filepath.Ext(path); ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var r *crawler.Rule
		err = yaml.Unmarshal(data, &r)
		if err != nil {
			return err
		}

		publishers[r.Name] = r

		return nil
	})
}

func List() map[string]Publisher {
	return publishers
}
