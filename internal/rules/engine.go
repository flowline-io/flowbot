package rules

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/rulego/rulego"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func InitEngine() error {
	conf, err := NewConfig()
	if err != nil {
		return err
	}

	_, err = rulego.New("test", utils.StringToBytes(testCustomDslYamlRule), rulego.WithConfig(conf))
	if err != nil {
		return err
	}
	flog.Info("load test rule")

	// load rules from directory

	rulesPath := config.App.Flowbot.RulesPath
	info, err := os.Stat(rulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			flog.Warn("The directory does not exist: %s", rulesPath)
			return nil // Ignore empty rules directory
		}
		return err
	}

	// Make sure it's a directory, not a file
	if !info.IsDir() {
		panic(fmt.Sprintf("The path is not a directory: %s", rulesPath))
	}

	var yamlFiles = make(map[string]string)

	// Traverse the directory
	err = filepath.WalkDir(rulesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle errors that may occur during traversal (e.g. permission issues)
			flog.Warn("Failed to access path %s: %v\n", path, err)
			return nil // Skip errors and continue traversal
		}

		if d.IsDir() {
			return nil // Skip directories
		}

		ext := strings.ToLower(filepath.Ext(path))

		ruleId, err := getRuleId(rulesPath, path, ext)
		if err != nil {
			return fmt.Errorf("get rule id error: %w", err)
		}

		if ext == ".yaml" || ext == ".yml" {
			yamlFiles[ruleId] = path
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("an error occurred while traversing the directory: %v", err)
	}

	for ruleId, yamlFile := range yamlFiles {
		go func(ruleId string, yamlFile string) {
			content, err := os.ReadFile(yamlFile)
			if err != nil {
				flog.Error(fmt.Errorf("load rule error: %w", err))
				return
			}
			_, err = rulego.New(ruleId, content, rulego.WithConfig(conf))
			if err != nil {
				flog.Error(fmt.Errorf("load rule error: %w", err))
				return
			}
			flog.Info("load %s rule", ruleId)
		}(ruleId, yamlFile)
	}

	// Watch the rules directory for changes
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			flog.Error(fmt.Errorf("failed to create watcher: %w", err))
			return
		}
		defer func() {
			_ = watcher.Close()
		}()

		// Watch the rules directory with subdirectories
		// add new directory, need restart app to watch new directory
		err = filepath.Walk(rulesPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if filepath.Base(path) == "." {
					return filepath.SkipDir
				}
				err = watcher.Add(path)
				if err != nil {
					return err
				}
				flog.Info("Watching directory: %s", path)
			}
			return nil
		})
		if err != nil {
			flog.Error(fmt.Errorf("failed to watch directory: %w", err))
			return
		}

		for {
			select {
			case event := <-watcher.Events:
				flog.Info("Rule File changed: %s", event.String())

				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext != ".yaml" && ext != ".yml" {
					continue
				}

				ruleId, err := getRuleId(rulesPath, event.Name, ext)
				if err != nil {
					flog.Error(fmt.Errorf("get rule id error: %w", err))
					return
				}

				if event.Has(fsnotify.Remove) {
					// Delete the rule
					rulego.Del(ruleId)
				}
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					def, err := os.ReadFile(event.Name)
					if err != nil {
						flog.Error(fmt.Errorf("read rule file error: %w", err))
						return
					}
					ruleEngine, ok := rulego.Get(ruleId)
					if !ok {
						// Load the rule
						_, err = rulego.New(ruleId, def, rulego.WithConfig(conf))
						if err != nil {
							flog.Error(fmt.Errorf("load rule error: %w", err))
						}
						return
					}
					// Reload the rule
					err = ruleEngine.ReloadSelf(def, rulego.WithConfig(conf))
					if err != nil {
						flog.Error(fmt.Errorf("reload rule error: %w", err))
						return
					}
				}
			case err := <-watcher.Errors:
				flog.Error(fmt.Errorf("watcher error: %w", err))
				return
			}
		}
	}()

	return nil
}

func getRuleId(rulesPath, path, ext string) (string, error) {
	relPath, err := filepath.Rel(rulesPath, path)
	if err != nil {
		return "", fmt.Errorf("an error occurred while getting the relative path: %v", err)
	}

	relPath = filepath.ToSlash(relPath)
	ruleId := strings.TrimSuffix(relPath, ext)

	return ruleId, nil
}
