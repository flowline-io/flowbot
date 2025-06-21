package script

import (
	"fmt"
	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (e *Engine) scan() error {
	// load scripts from directory

	scriptsPath := config.App.ScriptEngine.ScriptPath
	info, err := os.Stat(scriptsPath)
	if err != nil {
		if os.IsNotExist(err) {
			flog.Warn("The directory does not exist: %s", scriptsPath)
			return nil // Ignore empty scripts directory
		}
		return err
	}

	// Make sure it's a directory, not a file
	if !info.IsDir() {
		return fmt.Errorf("the path is not a directory: %s", scriptsPath)
	}

	var scriptFiles = make(map[string]string)

	// Traverse the directory
	err = filepath.WalkDir(scriptsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle errors that may occur during traversal (e.g. permission issues)
			flog.Warn("Failed to access path %s: %v\n", path, err)
			return nil // Skip errors and continue traversal
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		scriptId, err := getFileId(scriptsPath, path, ext)
		if err != nil {
			return fmt.Errorf("get rule id error: %w", err)
		}

		// only support bash and fish
		if ext == ".sh" || ext == ".fish" {
			scriptFiles[scriptId] = path
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("an error occurred while traversing the directory: %v", err)
	}

	for scriptId, path := range scriptFiles {
		flog.Info("load script: %s %s", scriptId, path)
	}

	// Watch scripts directory for changes
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
		err = filepath.Walk(scriptsPath, func(path string, info fs.FileInfo, err error) error {
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
				flog.Info("File %s has been %s", event.Name, event.Op.String())

				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext != ".sh" && ext != ".fish" {
					continue
				}

				scriptId, err := getFileId(scriptsPath, event.Name, ext)
				if err != nil {
					flog.Error(fmt.Errorf("get rule id error: %w", err))
					continue
				}

				flog.Info("load script: %s %s", scriptId, event.Name)

				if event.Op == fsnotify.Remove {
					// TODO: delete script
					flog.Info("delete script: %s", scriptId)
				}
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
					// TODO: reload script
					flog.Info("reload script: %s", scriptId)
				}
			case err := <-watcher.Errors:
				flog.Error(fmt.Errorf("watcher error: %w", err))
				return
			case <-e.stop:
				flog.Info("stop script engine's watcher")
				break
			}
		}
	}()

	return nil
}

func getFileId(rulesPath, path, ext string) (string, error) {
	relPath, err := filepath.Rel(rulesPath, path)
	if err != nil {
		return "", fmt.Errorf("an error occurred while getting the relative path: %v", err)
	}

	relPath = filepath.ToSlash(relPath)
	ruleId := strings.TrimSuffix(relPath, ext)

	return ruleId, nil
}
