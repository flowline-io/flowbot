// Package clip provides agent tools for creating and reading shareable markdown clips.
package clip

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/capability"
	capclip "github.com/flowline-io/flowbot/pkg/capability/clip"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	// CreateToolName is the agent tool name for creating clips.
	CreateToolName = "create_clip"
	// GetToolName is the agent tool name for reading clips by slug.
	GetToolName = "get_clip"
)

// AbsoluteURL builds a full clip URL from an optional public base and a relative path or slug.
func AbsoluteURL(publicBase, relativeOrSlug string) string {
	path := strings.TrimSpace(relativeOrSlug)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/c/" + path
	}
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(config.App.Flowbot.URL), "/")
	}
	if base == "" {
		return path
	}
	return base + path
}

// CreateTool creates a shareable markdown clip via the clip capability.
type CreateTool struct {
	// PublicBaseURL is the absolute site origin (e.g. https://flowbot.example.com).
	// When empty, config.App.Flowbot.URL is used.
	PublicBaseURL string
}

// Name returns the tool identifier.
func (CreateTool) Name() string { return CreateToolName }

// Description explains the tool to the model.
func (CreateTool) Description() string {
	return "Create a shareable markdown clip and return its full public URL"
}

// Parameters returns the JSON schema for tool arguments.
func (CreateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "Markdown body to store in the clip",
			},
			"created_by": map[string]any{
				"type":        "string",
				"description": "Optional creator identifier",
			},
		},
		"required": []string{"content"},
	}
}

// Execute creates a clip and returns the absolute public URL.
func (t CreateTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	content := strings.TrimSpace(fmt.Sprint(args["content"]))
	if content == "" || content == "<nil>" {
		return tool.ErrorResult(id, t.Name(), "invalid_args", "content is required", "pass the markdown body to publish"), nil
	}
	params := map[string]any{"content": content}
	if raw, ok := args["created_by"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(raw)); s != "" && s != "<nil>" {
			params["created_by"] = s
		}
	}

	res, err := capability.Invoke(ctx, hub.CapClip, capclip.OpCreate, params)
	if err != nil {
		return invokeErrorResult(id, t.Name(), err), nil
	}
	data := resultDataMap(res)
	slug := stringFromMap(data, "slug")
	relURL := stringFromMap(data, "url")
	if relURL == "" && slug != "" {
		relURL = "/c/" + slug
	}
	fullURL := AbsoluteURL(t.PublicBaseURL, relURL)
	title := stringFromMap(data, "title")
	text := fmt.Sprintf("clip created\nurl: %s\nslug: %s\ntitle: %s", fullURL, slug, title)
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		text += "\nnote: set flowbot.url for an absolute public link"
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       t.Name(),
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

// GetTool reads a shareable markdown clip by slug via the clip capability.
type GetTool struct{}

// Name returns the tool identifier.
func (GetTool) Name() string { return GetToolName }

// Description explains the tool to the model.
func (GetTool) Description() string {
	return "Read a shareable markdown clip by its slug"
}

// Parameters returns the JSON schema for tool arguments.
func (GetTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug": map[string]any{
				"type":        "string",
				"description": "Clip slug (for example KhpG3Hab from /c/KhpG3Hab)",
			},
		},
		"required": []string{"slug"},
	}
}

// Execute loads a clip and returns its metadata and markdown content.
func (GetTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	slug := strings.TrimSpace(fmt.Sprint(args["slug"]))
	if slug == "" || slug == "<nil>" {
		return tool.ErrorResult(id, GetToolName, "invalid_args", "slug is required", "pass the clip slug from /c/{slug}"), nil
	}

	res, err := capability.Invoke(ctx, hub.CapClip, capclip.OpGet, map[string]any{"slug": slug})
	if err != nil {
		return invokeErrorResult(id, GetToolName, err), nil
	}
	data := resultDataMap(res)
	title := stringFromMap(data, "title")
	description := stringFromMap(data, "description")
	content := stringFromMap(data, "content")
	relURL := stringFromMap(data, "url")
	text := fmt.Sprintf("slug: %s\ntitle: %s\ndescription: %s\nurl: %s\n\n%s",
		slug, title, description, AbsoluteURL("", relURL), content)
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       GetToolName,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}, nil
}

// Register registers create_clip and get_clip on the given registry.
func Register(registry *tool.Registry, publicBaseURL string) error {
	if registry == nil {
		return fmt.Errorf("clip tools: registry is nil")
	}
	tools := []tool.Tool{
		CreateTool{PublicBaseURL: publicBaseURL},
		GetTool{},
	}
	for _, item := range tools {
		if err := registry.Register(item); err != nil {
			return err
		}
	}
	return nil
}

// ActiveToolNames returns the default clip tool names.
func ActiveToolNames() []string {
	return []string{CreateToolName, GetToolName}
}

func invokeErrorResult(callID, name string, err error) msg.ToolResultMessage {
	code := "tool_error"
	hint := "retry or verify clip capability is registered"
	switch {
	case errors.Is(err, types.ErrNotFound):
		code = "not_found"
		hint = "check the slug and try again"
	case errors.Is(err, types.ErrInvalidArgument):
		code = "invalid_args"
		hint = "fix the tool arguments"
	case errors.Is(err, types.ErrUnavailable):
		code = "unavailable"
		hint = "ensure the clip capability persister is configured"
	}
	return tool.ErrorResult(callID, name, code, err.Error(), hint)
}

func resultDataMap(res *capability.InvokeResult) map[string]any {
	if res == nil {
		return nil
	}
	data, ok := res.Data.(map[string]any)
	if !ok {
		return nil
	}
	return data
}

func stringFromMap(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	raw, ok := data[key]
	if !ok || raw == nil {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return strings.TrimSpace(fmt.Sprint(raw))
	}
	return s
}
