package homelab

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var appNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Scanner struct {
	config Config
}

func NewScanner(config Config) *Scanner {
	return &Scanner{config: normalizeConfig(config)}
}

func (s *Scanner) Scan() ([]App, error) {
	appsDir, err := safeEval(s.config.AppsDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, fmt.Errorf("read apps dir: %w", err)
	}
	allowlist := allowlistSet(s.config.Allowlist)
	apps := make([]App, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		name := entry.Name()
		if !appNamePattern.MatchString(name) {
			continue
		}
		if len(allowlist) > 0 && !allowlist[name] {
			continue
		}
		app, err := s.scanApp(appsDir, name)
		if err != nil {
			return nil, err
		}
		if app.ComposeFile == "" {
			continue
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (s *Scanner) scanApp(appsDir, name string) (App, error) {
	path := filepath.Join(appsDir, name)
	realPath, err := safeEval(path)
	if err != nil {
		return App{}, err
	}
	if !isInside(appsDir, realPath) {
		return App{}, fmt.Errorf("app %s escapes apps dir", name)
	}
	composeFile := s.findComposeFile(realPath)
	if composeFile == "" {
		return App{Name: name, Path: realPath, Status: AppStatusUnknown, Health: HealthUnknown}, nil
	}
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return App{}, fmt.Errorf("read compose file: %w", err)
	}
	services, networks, ports, labels, err := ParseCompose(data)
	if err != nil {
		return App{}, err
	}
	return App{
		Name:        name,
		Path:        realPath,
		ComposeFile: composeFile,
		Services:    services,
		Networks:    networks,
		Ports:       ports,
		Labels:      labels,
		Status:      AppStatusUnknown,
		Health:      HealthUnknown,
	}, nil
}

func (s *Scanner) findComposeFile(path string) string {
	candidates := []string{"docker-compose.yaml", "compose.yaml"}
	if s.config.ComposeFile != "" {
		candidates = []string{s.config.ComposeFile}
	}
	for _, candidate := range candidates {
		file := filepath.Join(path, candidate)
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			return file
		}
	}
	return ""
}

func normalizeConfig(config Config) Config {
	if config.AppsDir == "" && config.Root != "" {
		config.AppsDir = filepath.Join(config.Root, "apps")
	}
	if config.Runtime.Mode == "" {
		config.Runtime.Mode = RuntimeModeNone
	}
	return config
}

func safeAbs(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return filepath.Clean(abs), nil
}

func safeEval(path string) (string, error) {
	abs, err := safeAbs(path)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("evaluate symlink: %w", err)
	}
	return filepath.Clean(realPath), nil
}

func isInside(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == "" || rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func allowlistSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		if value != "" {
			result[value] = true
		}
	}
	return result
}
