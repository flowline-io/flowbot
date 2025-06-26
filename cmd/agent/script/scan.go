package script

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/fsnotify/fsnotify"
	"github.com/samber/lo"
)

func (e *Engine) scan() error {
	// load scripts from directory

	scriptsPath := config.App.ScriptEngine.ScriptPath
	info, err := os.Stat(scriptsPath)
	if err != nil {
		if os.IsNotExist(err) {
			flog.Warn("[script] The directory does not exist: %s", scriptsPath)
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
			flog.Warn("[script] Failed to access path %s: %v\n", path, err)
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
		// load script
		rule, err := parseScript(scriptId, path)
		if err != nil {
			flog.Error(fmt.Errorf("[script] failed to parse script: %w", err))
			continue
		}
		err = e.loadScriptJob(context.Background(), rule)
		if err != nil {
			flog.Error(fmt.Errorf("[script] failed to load script: %w", err))
			continue
		}
		flog.Info("[script] load script: %s %s", scriptId, path)
	}

	// Watch scripts directory for changes
	go e.watcher()

	return nil
}

func (e *Engine) watcher() {
	scriptsPath := config.App.ScriptEngine.ScriptPath
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		flog.Error(fmt.Errorf("[script] failed to create watcher: %w", err))
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
			dir := filepath.Base(path)
			if dir == "." {
				return filepath.SkipDir
			}
			if lo.Count(config.App.ScriptEngine.SkipWatchDir, dir) > 0 {
				return filepath.SkipDir
			}
			err = watcher.Add(path)
			if err != nil {
				return err
			}
			flog.Info("[script] Watching directory: %s", path)
		}
		return nil
	})
	if err != nil {
		flog.Error(fmt.Errorf("[script] failed to watch directory: %w", err))
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			flog.Info("[script] File %s has been %s", event.Name, event.Op.String())

			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext != ".sh" && ext != ".fish" {
				continue
			}

			scriptId, err := getFileId(scriptsPath, event.Name, ext)
			if err != nil {
				flog.Error(fmt.Errorf("[script] get rule id error: %w", err))
				continue
			}

			if event.Has(fsnotify.Create) {
				// load script
				rule, err := parseScript(scriptId, event.Name)
				if err != nil {
					flog.Error(err)
					continue
				}
				err = e.loadScriptJob(context.Background(), rule)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[script] load script: %s", scriptId)
			}
			if event.Has(fsnotify.Rename) {
				// delete script
				rule := Rule{
					Id: scriptId,
				}
				err = e.deleteScriptJob(context.Background(), rule)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[script] delete script: %s", scriptId)
			}
			if event.Has(fsnotify.Write) {
				// reload script
				rule, err := parseScript(scriptId, event.Name)
				if err != nil {
					flog.Error(err)
					continue
				}
				err = e.reloadScriptJob(context.Background(), rule)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[script] reload script: %s", scriptId)
			}
			if event.Op == fsnotify.Remove {
				// delete script
				rule := Rule{
					Id: scriptId,
				}
				err = e.deleteScriptJob(context.Background(), rule)
				if err != nil {
					flog.Error(err)
				}
				flog.Info("[script] delete script: %s", scriptId)
			}
		case err := <-watcher.Errors:
			flog.Error(fmt.Errorf("[script] watcher error: %w", err))
		case <-e.stop:
			flog.Info("[script] stop script engine's watcher")
			return
		}
	}
}

func getFileId(rulesPath, path, ext string) (string, error) {
	relPath, err := filepath.Rel(rulesPath, path)
	if err != nil {
		return "", fmt.Errorf("an error occurred while getting the relative path: %w", err)
	}

	relPath = filepath.ToSlash(relPath)
	ruleId := strings.TrimSuffix(relPath, ext)

	return ruleId, nil
}

func (e *Engine) loadScriptJob(ctx context.Context, r Rule) error {
	if r.When != "" {
		_, err := e.addCronJob(r)
		return err
	} else {
		return e.pushQueue(ctx, r)
	}
}

func (e *Engine) deleteScriptJob(_ context.Context, r Rule) error {
	if r.When != "" {
		e.removeCronJob(r)
	}
	return nil
}

func (e *Engine) reloadScriptJob(ctx context.Context, r Rule) error {
	if r.When != "" {
		err := e.deleteScriptJob(ctx, r)
		if err != nil {
			return err
		}
	}
	return e.loadScriptJob(ctx, r)
}

func parseScript(scriptId, path string) (Rule, error) {
	scriptContent, err := os.ReadFile(path)
	if err != nil {
		return Rule{}, fmt.Errorf("failed to read script file: %w", err)
	}
	metadata, err := parseMetadata(scriptContent)
	if err != nil {
		return Rule{}, fmt.Errorf("failed to parse metadata: %w", err)
	}
	flog.Info("[script] %s script metadata: %#v", scriptId, metadata)

	r := Rule{
		Id:         scriptId,
		Path:       path,
		Timeout:    time.Hour,
		When:       metadata[cronMetadataTag],
		Version:    metadata[versionMetadataTag],
		Desciption: metadata[descriptionMetadataTag],
	}

	if v, ok := metadata[timeoutMetadataTag]; ok {
		timeout, err := time.ParseDuration(v)
		if err != nil {
			return Rule{}, fmt.Errorf("failed to parse timeout: %w", err)
		}
		r.Timeout = timeout
	}
	if v, ok := metadata[retriesMetadataTag]; ok {
		retries, err := strconv.Atoi(v)
		if err != nil {
			return Rule{}, fmt.Errorf("failed to parse retries: %w", err)
		}
		r.Retries = retries
	}

	return r, nil
}

const (
	cronMetadataTag        = "cron"
	timeoutMetadataTag     = "timeout"
	versionMetadataTag     = "version"
	descriptionMetadataTag = "description"
	retriesMetadataTag     = "retries"
)

func parseMetadata(scriptContent []byte) (map[string]string, error) {
	metadata := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewReader(scriptContent))

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		content := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "#"))

		if !strings.HasPrefix(content, "@") {
			continue
		}

		parts := strings.SplitN(content, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], "@")

		value := strings.TrimSpace(parts[1])
		processedValue := strings.Trim(value, `"`)

		metadata[key] = processedValue
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading script error: %w", err)
	}

	return metadata, nil
}
