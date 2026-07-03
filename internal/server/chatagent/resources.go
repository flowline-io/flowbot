package chatagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// PlanLocationPrefix is the URI scheme for persisted plan documents.
	PlanLocationPrefix = "plan://"
	// FileLocationPrefix is the URI scheme for workspace-relative files.
	FileLocationPrefix = "file://"
)

var resourceURIPattern = regexp.MustCompile(`\[[^\]]*\]\((plan://[^)]+|file://[^)]+)\)`)

// ResourceContent is the resolved body of a chat agent resource URI.
type ResourceContent struct {
	URI         string
	Kind        string
	Title       string
	Content     string
	ContentType string
	Truncated   bool
}

// ParseResourceURI splits a resource URI into scheme and reference.
func ParseResourceURI(uri string) (scheme, ref string, err error) {
	uri = strings.TrimSpace(uri)
	switch {
	case strings.HasPrefix(uri, PlanLocationPrefix):
		ref = strings.TrimSpace(strings.TrimPrefix(uri, PlanLocationPrefix))
		if ref == "" {
			return "", "", types.Errorf(types.ErrInvalidArgument, "plan id is required")
		}
		return "plan", ref, nil
	case strings.HasPrefix(uri, FileLocationPrefix):
		ref = strings.TrimSpace(strings.TrimPrefix(uri, FileLocationPrefix))
		if ref == "" {
			return "", "", types.Errorf(types.ErrInvalidArgument, "file path is required")
		}
		return "file", filepath.ToSlash(filepath.Clean(ref)), nil
	default:
		return "", "", types.Errorf(types.ErrInvalidArgument, "unsupported resource URI: %q", uri)
	}
}

// ResolveResource loads one plan:// or file:// resource for a session.
func ResolveResource(ctx context.Context, sessionID, uri string) (ResourceContent, error) {
	if strings.TrimSpace(sessionID) == "" {
		return ResourceContent{}, types.Errorf(types.ErrInvalidArgument, "session_id is required")
	}
	scheme, ref, err := ParseResourceURI(uri)
	if err != nil {
		return ResourceContent{}, err
	}
	switch scheme {
	case "plan":
		return resolvePlanResource(ctx, sessionID, uri, ref)
	case "file":
		return resolveFileResource(uri, ref)
	default:
		return ResourceContent{}, types.Errorf(types.ErrInvalidArgument, "unsupported resource scheme: %q", scheme)
	}
}

func resolvePlanResource(ctx context.Context, sessionID, uri, planID string) (ResourceContent, error) {
	if store.Database == nil {
		return ResourceContent{}, types.ErrUnavailable
	}
	row, err := store.Database.GetAgentPlanInSession(ctx, sessionID, planID)
	if err != nil {
		return ResourceContent{}, err
	}
	return ResourceContent{
		URI:         uri,
		Kind:        "plan",
		Title:       row.Title,
		Content:     row.Content,
		ContentType: "text/markdown",
	}, nil
}

func resolveFileResource(uri, relPath string) (ResourceContent, error) {
	ws, err := WorkspaceFromConfig()
	if err != nil {
		return ResourceContent{}, err
	}
	resolved := ws.ResolvePath(relPath)
	if !resolved.IsOk() {
		return ResourceContent{}, types.Errorf(types.ErrForbidden, "%s", env.FormatFileError(resolved.ErrorValue()))
	}
	data, err := os.ReadFile(resolved.Value())
	if err != nil {
		if os.IsNotExist(err) {
			return ResourceContent{}, types.ErrNotFound
		}
		return ResourceContent{}, fmt.Errorf("read file: %w", err)
	}
	if !utf8.Valid(data) {
		return ResourceContent{}, types.Errorf(types.ErrInvalidArgument, "file is not valid UTF-8 text")
	}
	raw := string(data)
	content := ws.TruncateOutput(raw)
	truncated := strings.HasSuffix(content, "\n...(output truncated)")
	title := filepath.Base(relPath)
	contentType := "text/plain"
	if strings.HasSuffix(strings.ToLower(relPath), ".md") {
		contentType = "text/markdown"
	}
	return ResourceContent{
		URI:         uri,
		Kind:        "file",
		Title:       title,
		Content:     content,
		ContentType: contentType,
		Truncated:   truncated,
	}, nil
}

// ExtractResourceURIs returns plan:// and file:// URIs referenced in markdown links.
func ExtractResourceURIs(text string) []string {
	matches := resourceURIPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	uris := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		uri := strings.TrimSpace(match[1])
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		uris = append(uris, uri)
	}
	return uris
}
