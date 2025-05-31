package rules

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
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

		relPath, err := filepath.Rel(rulesPath, path)
		if err != nil {
			return fmt.Errorf("an error occurred while getting the relative path: %v", err)
		}

		relPath = filepath.ToSlash(relPath)
		ruleId := strings.TrimSuffix(relPath, ext)

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
		}(ruleId, yamlFile)
	}

	return nil
}
