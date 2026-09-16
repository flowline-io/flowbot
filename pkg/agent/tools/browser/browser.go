// Package browser registers chat-agent browser_* tools backed by pkg/agent/browser.
package browser

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	agentbrowser "github.com/flowline-io/flowbot/pkg/agent/browser"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

const (
	ToolNavigate   = "browser_navigate"
	ToolSnapshot   = "browser_snapshot"
	ToolClick      = "browser_click"
	ToolType       = "browser_type"
	ToolScroll     = "browser_scroll"
	ToolWait       = "browser_wait"
	ToolScreenshot = "browser_screenshot"
)

// ActiveToolNames returns browser tool names when registered.
func ActiveToolNames() []string {
	return []string{
		ToolNavigate,
		ToolSnapshot,
		ToolClick,
		ToolType,
		ToolScroll,
		ToolWait,
		ToolScreenshot,
	}
}

// Register adds browser tools to the registry.
func Register(registry *tool.Registry) error {
	tools := []tool.Tool{
		NavigateTool{},
		SnapshotTool{},
		ClickTool{},
		TypeTool{},
		ScrollTool{},
		WaitTool{},
		ScreenshotTool{},
	}
	for _, t := range tools {
		if err := registry.Register(t); err != nil {
			return err
		}
	}
	return nil
}

func sessionOrError(ctx context.Context, id, name string) (*agentbrowser.Session, *msg.ToolResultMessage) {
	sess, ok := agentbrowser.FromContext(ctx)
	if !ok {
		res := tool.ErrorResult(id, name, "browser_unavailable", "browser session is not attached to this run", "enable chat_agent.browser and use the chat agent")
		return nil, &res
	}
	return sess, nil
}

func okResult(id, name, text string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: text}},
	}
}

func toolErr(id, name string, err error) msg.ToolResultMessage {
	return tool.ErrorResult(id, name, "browser_error", err.Error(), "retry after browser_snapshot if the page changed")
}

// NavigateTool loads a URL in the run browser context.
type NavigateTool struct{}

// Name returns the tool identifier.
func (NavigateTool) Name() string { return ToolNavigate }

// Description explains the tool to the model.
func (NavigateTool) Description() string {
	return "Navigate the browser to an http(s) URL; private/metadata hosts blocked unless allowlisted"
}

// Parameters returns the JSON schema for tool arguments.
func (NavigateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{"type": "string", "description": "Absolute http(s) URL"},
		},
		"required": []string{"url"},
	}
}

// Execute navigates to the URL.
func (NavigateTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolNavigate)
	if errRes != nil {
		return *errRes, nil
	}
	rawURL := strings.TrimSpace(fmt.Sprint(args["url"]))
	final, err := sess.Navigate(ctx, rawURL)
	if err != nil {
		return toolErr(id, ToolNavigate, err), nil
	}
	if final == "" {
		final = rawURL
	}
	return okResult(id, ToolNavigate, "navigated to "+final), nil
}

// SnapshotTool returns a compact interactive tree with refs.
type SnapshotTool struct{}

// Name returns the tool identifier.
func (SnapshotTool) Name() string { return ToolSnapshot }

// Description explains the tool to the model.
func (SnapshotTool) Description() string {
	return "Capture an interactive page snapshot with opaque refs for browser_click and browser_type"
}

// Parameters returns the JSON schema for tool arguments.
func (SnapshotTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Execute returns the snapshot text.
func (SnapshotTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolSnapshot)
	if errRes != nil {
		return *errRes, nil
	}
	tree, err := sess.Snapshot(ctx)
	if err != nil {
		return toolErr(id, ToolSnapshot, err), nil
	}
	return okResult(id, ToolSnapshot, tree), nil
}

// ClickTool clicks a snapshot ref.
type ClickTool struct{}

// Name returns the tool identifier.
func (ClickTool) Name() string { return ToolClick }

// Description explains the tool to the model.
func (ClickTool) Description() string {
	return "Click an element by ref from the latest browser_snapshot"
}

// Parameters returns the JSON schema for tool arguments.
func (ClickTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ref": map[string]any{"type": "string", "description": "Opaque ref from browser_snapshot"},
		},
		"required": []string{"ref"},
	}
}

// Execute clicks the ref.
func (ClickTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolClick)
	if errRes != nil {
		return *errRes, nil
	}
	ref := strings.TrimSpace(fmt.Sprint(args["ref"]))
	if err := sess.Click(ctx, ref); err != nil {
		return toolErr(id, ToolClick, err), nil
	}
	return okResult(id, ToolClick, "clicked "+ref), nil
}

// TypeTool types into a snapshot ref.
type TypeTool struct{}

// Name returns the tool identifier.
func (TypeTool) Name() string { return ToolType }

// Description explains the tool to the model.
func (TypeTool) Description() string {
	return "Type text into an element by ref from the latest browser_snapshot"
}

// Parameters returns the JSON schema for tool arguments.
func (TypeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ref":  map[string]any{"type": "string", "description": "Opaque ref from browser_snapshot"},
			"text": map[string]any{"type": "string", "description": "Text to type"},
		},
		"required": []string{"ref", "text"},
	}
}

// Execute types into the ref.
func (TypeTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolType)
	if errRes != nil {
		return *errRes, nil
	}
	ref := stringArg(args, "ref")
	text := stringArg(args, "text")
	if err := sess.Type(ctx, ref, text); err != nil {
		return toolErr(id, ToolType, err), nil
	}
	return okResult(id, ToolType, fmt.Sprintf("typed %d chars into %s", len(text), ref)), nil
}

// ScrollTool scrolls the page.
type ScrollTool struct{}

// Name returns the tool identifier.
func (ScrollTool) Name() string { return ToolScroll }

// Description explains the tool to the model.
func (ScrollTool) Description() string {
	return "Scroll the page by pixel deltas"
}

// Parameters returns the JSON schema for tool arguments.
func (ScrollTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dx": map[string]any{"type": "integer", "description": "Horizontal scroll delta in pixels"},
			"dy": map[string]any{"type": "integer", "description": "Vertical scroll delta in pixels"},
		},
	}
}

// Execute scrolls the page.
func (ScrollTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolScroll)
	if errRes != nil {
		return *errRes, nil
	}
	dx, err := intArg(args, "dx")
	if err != nil {
		return tool.ErrorResult(id, ToolScroll, "invalid_args", err.Error(), "provide integer dx/dy"), nil
	}
	dy, err := intArg(args, "dy")
	if err != nil {
		return tool.ErrorResult(id, ToolScroll, "invalid_args", err.Error(), "provide integer dx/dy"), nil
	}
	if dx == 0 && dy == 0 {
		return tool.ErrorResult(id, ToolScroll, "invalid_args", "dx or dy is required", "provide a non-zero scroll delta"), nil
	}
	if err := sess.Scroll(ctx, dx, dy); err != nil {
		return toolErr(id, ToolScroll, err), nil
	}
	return okResult(id, ToolScroll, fmt.Sprintf("scrolled dx=%d dy=%d", dx, dy)), nil
}

// WaitTool waits for a short duration.
type WaitTool struct{}

// Name returns the tool identifier.
func (WaitTool) Name() string { return ToolWait }

// Description explains the tool to the model.
func (WaitTool) Description() string {
	return "Wait for milliseconds to allow the page to settle"
}

// Parameters returns the JSON schema for tool arguments.
func (WaitTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ms": map[string]any{"type": "integer", "description": "Milliseconds to wait (default 500)"},
		},
	}
}

// Execute waits.
func (WaitTool) Execute(ctx context.Context, id string, args map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolWait)
	if errRes != nil {
		return *errRes, nil
	}
	ms, err := intArg(args, "ms")
	if err != nil {
		return tool.ErrorResult(id, ToolWait, "invalid_args", err.Error(), "provide integer ms"), nil
	}
	if err := sess.Wait(ctx, ms); err != nil {
		return toolErr(id, ToolWait, err), nil
	}
	if ms <= 0 {
		ms = 500
	}
	return okResult(id, ToolWait, fmt.Sprintf("waited %dms", ms)), nil
}

// ScreenshotTool captures a PNG of the page.
type ScreenshotTool struct{}

// Name returns the tool identifier.
func (ScreenshotTool) Name() string { return ToolScreenshot }

// Description explains the tool to the model.
func (ScreenshotTool) Description() string {
	return "Capture a PNG screenshot of the current page for visual inspection"
}

// Parameters returns the JSON schema for tool arguments.
func (ScreenshotTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// Execute captures a screenshot as image media plus text metadata.
func (ScreenshotTool) Execute(ctx context.Context, id string, _ map[string]any, _ tool.UpdateHandler) (msg.ToolResultMessage, error) {
	sess, errRes := sessionOrError(ctx, id, ToolScreenshot)
	if errRes != nil {
		return *errRes, nil
	}
	png, err := sess.Screenshot(ctx)
	if err != nil {
		return toolErr(id, ToolScreenshot, err), nil
	}
	return msg.ToolResultMessage{
		ToolCallID: id,
		Name:       ToolScreenshot,
		Parts: []msg.ContentPart{
			msg.TextPart{Text: fmt.Sprintf("screenshot png bytes=%d", len(png))},
			msg.MediaPart{Kind: msg.MediaKindImage, MIMEType: "image/png", Data: png},
		},
	}, nil
}

func stringArg(args map[string]any, key string) string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func intArg(args map[string]any, key string) (int, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0, nil
	}
	switch v := raw.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}
