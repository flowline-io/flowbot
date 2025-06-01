package rules

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/fsnotify/fsnotify"
	endpointTypes "github.com/rulego/rulego/api/types/endpoint"
	"github.com/rulego/rulego/endpoint"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const EndpointDirName = "endpoints"

func newEndpoint(id string, def []byte) (endpointTypes.DynamicEndpoint, error) {
	conf, err := NewConfig()
	if err != nil {
		return nil, err
	}

	return endpoint.New(id, def, endpointTypes.DynamicEndpointOptions.WithConfig(conf))
}

func reloadEndpoint(endpoint endpointTypes.DynamicEndpoint, def []byte) error {
	conf, err := NewConfig()
	if err != nil {
		return err
	}

	return endpoint.Reload(def, endpointTypes.DynamicEndpointOptions.WithConfig(conf))
}

func InitEndpoint() error {
	// load endpoints from directory

	rulesPath := config.App.Flowbot.RulesPath
	endpointsPath := filepath.Join(rulesPath, EndpointDirName)
	info, err := os.Stat(endpointsPath)
	if err != nil {
		if os.IsNotExist(err) {
			flog.Warn("The directory does not exist: %s", endpointsPath)
			return nil // Ignore empty rules directory
		}
		return err
	}

	// Make sure it's a directory, not a file
	if !info.IsDir() {
		return fmt.Errorf("the path is not a directory: %s", endpointsPath)
	}

	var yamlFiles = make(map[string]string)

	// Traverse the directory
	err = filepath.WalkDir(endpointsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle errors that may occur during traversal (e.g. permission issues)
			flog.Warn("Failed to access path %s: %v\n", path, err)
			return nil // Skip errors and continue traversal
		}

		if d.IsDir() {
			return nil // Skip directories
		}

		ext := strings.ToLower(filepath.Ext(path))

		endpointId, err := getFileId(endpointsPath, path, ext)
		if err != nil {
			return fmt.Errorf("get rule id error: %w", err)
		}

		if ext == ".yaml" || ext == ".yml" {
			yamlFiles[endpointId] = path
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("an error occurred while traversing the directory: %v", err)
	}

	for endpointId, yamlFile := range yamlFiles {
		go func(endpointId string, yamlFile string) {
			content, err := os.ReadFile(yamlFile)
			if err != nil {
				flog.Error(fmt.Errorf("read endpoint file error: %w", err))
				return
			}

			content, err = utils.YamlToJson(content)
			if err != nil {
				flog.Error(fmt.Errorf("yaml to json error: %w", err))
				return
			}
			ep, err := newEndpoint(endpointId, content)
			if err != nil {
				flog.Error(fmt.Errorf("load endpoint error: %w", err))
				return
			}
			flog.Info("load %s endpoint", endpointId)

			err = ep.Start()
			if err != nil {
				flog.Error(fmt.Errorf("start endpoint error: %w", err))
			}
		}(endpointId, yamlFile)
	}

	// Watch the endpoints directory for changes
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			flog.Error(fmt.Errorf("failed to create watcher: %w", err))
			return
		}
		defer func() {
			_ = watcher.Close()
		}()

		err = watcher.Add(endpointsPath)
		if err != nil {
			flog.Error(fmt.Errorf("failed to watch directory: %w", err))
			return
		}
		flog.Info("Watching directory: %s", endpointsPath)

		for {
			select {
			case event := <-watcher.Events:
				flog.Info("Endpoint File changed: %s", event.String())

				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext != ".yaml" && ext != ".yml" {
					continue
				}

				endpointId, err := getFileId(endpointsPath, event.Name, ext)
				if err != nil {
					flog.Error(fmt.Errorf("get endpoint id error: %w", err))
					return
				}

				if event.Has(fsnotify.Remove) {
					// Delete the endpoint
					endpoint.Del(endpointId)
				}
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					def, err := os.ReadFile(event.Name)
					if err != nil {
						flog.Error(fmt.Errorf("read endpoint file error: %w", err))
						return
					}
					def, err = utils.YamlToJson(def)
					if err != nil {
						flog.Error(fmt.Errorf("yaml to json error: %w", err))
						return
					}
					ep, ok := endpoint.Get(endpointId)
					if !ok {
						// Load the endpoint
						go func(endpointId string, def []byte) {
							ep, err = newEndpoint(endpointId, def)
							if err != nil {
								flog.Error(fmt.Errorf("load endpoint error: %w", err))
							}

							err = ep.Start()
							if err != nil {
								flog.Error(fmt.Errorf("start endpoint error: %w", err))
							}
						}(endpointId, def)
						return
					}
					// Reload the endpoint
					err = reloadEndpoint(ep, def)
					if err != nil {
						flog.Error(fmt.Errorf("reload endpoint error: %w", err))
						return
					}
				}
			case err := <-watcher.Errors:
				flog.Error(fmt.Errorf("watcher error: %w", err))
			}
		}
	}()

	return nil
}
