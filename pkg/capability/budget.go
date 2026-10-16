package capability

import "context"

// DefaultMaxOutputBytes is the MCP tool result cap when mcp.max_output_bytes is unset or zero.
const DefaultMaxOutputBytes = 32768

// StubItemMessage is the _truncated value of an item that could not be shrunk under the output cap.
const StubItemMessage = "item exceeds mcp.max_output_bytes"

type pageBudgetKey struct{}

// PageBudget is the MCP output cap for the current call.
// Measure returns the UTF-8 size of the tool JSON after trace is attached.
type PageBudget struct {
	Max     int
	Measure func(items []any, page map[string]any) (int, error)
}

type cursorSecretKey struct{}

type cursorSecretBox struct {
	secret []byte
}

// WithPageBudget attaches an output cap to ctx. A non-positive Max disables clamping.
func WithPageBudget(ctx context.Context, budget PageBudget) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, pageBudgetKey{}, budget)
}

func pageBudgetFrom(ctx context.Context) (PageBudget, bool) {
	if ctx == nil {
		return PageBudget{}, false
	}
	budget, ok := ctx.Value(pageBudgetKey{}).(PageBudget)
	if !ok || budget.Max <= 0 || budget.Measure == nil {
		return PageBudget{}, false
	}
	return budget, true
}

// WithCursorSecretBox returns ctx carrying a box list handlers can fill with the HMAC secret
// that signed page.next_cursor. The box is a pointer so the value is visible to the caller.
func WithCursorSecretBox(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(cursorSecretKey{}).(*cursorSecretBox); ok {
		return ctx
	}
	return context.WithValue(ctx, cursorSecretKey{}, &cursorSecretBox{})
}

// PublishCursorSecret records secret for a later clamp on this call.
// A nil box (callers outside MCP) is ignored.
func PublishCursorSecret(ctx context.Context, secret []byte) {
	if ctx == nil || len(secret) == 0 {
		return
	}
	box, ok := ctx.Value(cursorSecretKey{}).(*cursorSecretBox)
	if !ok || box == nil {
		return
	}
	box.secret = append([]byte(nil), secret...)
}

func publishedCursorSecret(ctx context.Context) []byte {
	if ctx == nil {
		return nil
	}
	box, ok := ctx.Value(cursorSecretKey{}).(*cursorSecretBox)
	if !ok || box == nil || len(box.secret) == 0 {
		return nil
	}
	return box.secret
}
