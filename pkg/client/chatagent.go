package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	sseDialTimeout       = 10 * time.Second
	sseHeaderReadTimeout = 30 * time.Second
)

// ChatAgentClient calls the Chat Agent HTTP API.
type ChatAgentClient struct {
	c *Client
}

// ChatAgentInfo is splash metadata from GET /chatagent/info.
type ChatAgentInfo struct {
	Version    string          `json:"version"`
	ChatModel  string          `json:"chat_model"`
	ToolModel  string          `json:"tool_model"`
	Provider   string          `json:"provider"`
	Workspace  string          `json:"workspace"`
	Tools      []ChatToolInfo  `json:"tools"`
	Skills     []ChatSkillInfo `json:"skills"`
	ToolCount  int             `json:"tool_count"`
	SkillCount int             `json:"skill_count"`
}

// ChatToolInfo describes one active tool.
type ChatToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ChatSkillInfo describes one enabled skill.
type ChatSkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ChatContextCategory is one row in the context usage breakdown.
type ChatContextCategory struct {
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Tokens  int     `json:"tokens"`
	Percent float64 `json:"percent"`
}

// ChatContextSkill reports estimated prompt tokens for one skill entry.
type ChatContextSkill struct {
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
}

// ChatContextUsage is the context budget snapshot from GET /chatagent/sessions/:id/context.
type ChatContextUsage struct {
	Model             string                `json:"model"`
	ToolModel         string                `json:"tool_model,omitempty"`
	ContextWindow     int                   `json:"context_window"`
	TotalTokens       int                   `json:"total_tokens"`
	TotalPercent      float64               `json:"total_percent"`
	CompactionEnabled bool                  `json:"compaction_enabled"`
	Categories        []ChatContextCategory `json:"categories"`
	Skills            []ChatContextSkill    `json:"skills"`
}

// ChatCompactionResult is the outcome of POST /chatagent/sessions/:id/compact.
type ChatCompactionResult struct {
	Compacted    bool `json:"compacted"`
	TokensBefore int  `json:"tokens_before"`
	TokensAfter  int  `json:"tokens_after"`
}

// ChatHistoryMessage is one persisted chat turn.
type ChatHistoryMessage struct {
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatSessionSummary is one row from GET /chatagent/sessions.
type ChatSessionSummary struct {
	SessionID string    `json:"session_id"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatSessionExport is the full session snapshot from GET /chatagent/sessions/:id/export.
type ChatSessionExport struct {
	SessionID  string           `json:"session_id"`
	UID        string           `json:"uid"`
	LeafID     string           `json:"leaf_id"`
	State      string           `json:"state"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	ExportedAt time.Time        `json:"exported_at"`
	EntryCount int              `json:"entry_count"`
	Entries    []map[string]any `json:"entries"`
}

// ChatStreamEvent is one SSE payload from POST /chatagent/sessions/:id/messages.
type ChatStreamEvent struct {
	Type string `json:"type"`

	Text    string `json:"text,omitempty"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status,omitempty"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Message string `json:"message,omitempty"`

	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	ContextPercent   float64 `json:"context_percent,omitempty"`
	ContextWindow    int     `json:"context_window,omitempty"`

	ID       string `json:"id,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Approved bool   `json:"approved,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// Info returns Chat Agent splash metadata.
func (cc *ChatAgentClient) Info(ctx context.Context) (*ChatAgentInfo, error) {
	var info ChatAgentInfo
	if err := cc.chatGet(ctx, "/chatagent/info", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// CreateSession starts a new chat session.
func (cc *ChatAgentClient) CreateSession(ctx context.Context) (string, error) {
	var resp struct {
		SessionID string `json:"session_id"`
	}
	if err := cc.chatPost(ctx, "/chatagent/sessions", map[string]any{}, &resp); err != nil {
		return "", err
	}
	return resp.SessionID, nil
}

// CloseSession ends a chat session.
func (cc *ChatAgentClient) CloseSession(ctx context.Context, sessionID string) error {
	return cc.chatDelete(ctx, "/chatagent/sessions/"+sessionID)
}

// ListSessions returns active sessions owned by the authenticated user.
func (cc *ChatAgentClient) ListSessions(ctx context.Context, cursor string, limit int) ([]ChatSessionSummary, string, error) {
	path := "/chatagent/sessions"
	sep := "?"
	if limit > 0 {
		path += fmt.Sprintf("%slimit=%d", sep, limit)
		sep = "&"
	}
	if cursor != "" {
		path += sep + "cursor=" + cursor
	}
	var resp struct {
		Sessions []ChatSessionSummary `json:"sessions"`
		Cursor   string               `json:"cursor"`
	}
	if err := cc.chatGet(ctx, path, &resp); err != nil {
		return nil, "", err
	}
	return resp.Sessions, resp.Cursor, nil
}

// ListMessages returns persisted session messages.
func (cc *ChatAgentClient) ListMessages(ctx context.Context, sessionID string) ([]ChatHistoryMessage, error) {
	var resp struct {
		Messages []ChatHistoryMessage `json:"messages"`
	}
	if err := cc.chatGet(ctx, "/chatagent/sessions/"+sessionID+"/messages", &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

// ExportSession returns the full persisted session tree from the server.
func (cc *ChatAgentClient) ExportSession(ctx context.Context, sessionID string) (*ChatSessionExport, error) {
	var export ChatSessionExport
	if err := cc.chatGet(ctx, "/chatagent/sessions/"+sessionID+"/export", &export); err != nil {
		return nil, err
	}
	return &export, nil
}

// ContextUsage returns the estimated context budget breakdown for a session.
func (cc *ChatAgentClient) ContextUsage(ctx context.Context, sessionID string) (*ChatContextUsage, error) {
	var usage ChatContextUsage
	if err := cc.chatGet(ctx, "/chatagent/sessions/"+sessionID+"/context", &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// Compact triggers manual compaction for the current session branch.
func (cc *ChatAgentClient) Compact(ctx context.Context, sessionID string) (*ChatCompactionResult, error) {
	var result ChatCompactionResult
	if err := cc.chatPost(ctx, "/chatagent/sessions/"+sessionID+"/compact", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Confirm responds to a pending tool confirmation.
func (cc *ChatAgentClient) Confirm(ctx context.Context, sessionID, confirmID string, approved bool) error {
	return cc.chatPost(ctx, "/chatagent/sessions/"+sessionID+"/confirm", map[string]any{
		"id":       confirmID,
		"approved": approved,
	}, nil)
}

// Cancel aborts the in-flight run for a session.
func (cc *ChatAgentClient) Cancel(ctx context.Context, sessionID string) error {
	return cc.chatPost(ctx, "/chatagent/sessions/"+sessionID+"/cancel", map[string]any{}, nil)
}

func (cc *ChatAgentClient) chatGet(ctx context.Context, path string, result any) error {
	resp, err := cc.c.rc.R().SetContext(ctx).Get(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseChatResponse(resp.StatusCode(), resp.Bytes(), result)
}

func (cc *ChatAgentClient) chatPost(ctx context.Context, path string, body, result any) error {
	resp, err := cc.c.rc.R().SetContext(ctx).SetBody(body).Post(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseChatResponse(resp.StatusCode(), resp.Bytes(), result)
}

func (cc *ChatAgentClient) chatDelete(ctx context.Context, path string) error {
	resp, err := cc.c.rc.R().SetContext(ctx).Delete(path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	return parseChatResponse(resp.StatusCode(), resp.Bytes(), nil)
}

func parseChatResponse(status int, body []byte, result any) error {
	if status >= 300 {
		msg := string(body)
		var errBody struct {
			Error string `json:"error"`
		}
		if sonic.Unmarshal(body, &errBody) == nil && errBody.Error != "" {
			msg = errBody.Error
		}
		return &APIError{StatusCode: status, Message: msg}
	}
	if result == nil || len(body) == 0 || status == http.StatusNoContent {
		return nil
	}
	if err := sonic.Unmarshal(body, result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// SendMessageSSE streams one user turn and invokes onEvent for each SSE frame.
func (cc *ChatAgentClient) SendMessageSSE(ctx context.Context, sessionID, text string, onEvent func(ChatStreamEvent) error) error {
	body, err := sonic.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(cc.c.baseURL, "/")+"/chatagent/sessions/"+sessionID+"/messages",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AccessToken", cc.c.token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := sseHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("sse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}

	return readSSE(resp.Body, onEvent)
}

// sseHTTPClient returns an HTTP client suitable for long-lived SSE streams.
// Dial and response-header waits are bounded; the response body has no overall timeout.
func sseHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   sseDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: sseHeaderReadTimeout,
		},
	}
}

func readSSE(r io.Reader, onEvent func(ChatStreamEvent) error) error {
	scanner := bufio.NewScanner(r)
	var dataLines []string

	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = nil
		if payload == "" {
			return nil
		}
		var event ChatStreamEvent
		if err := sonic.UnmarshalString(payload, &event); err != nil {
			return fmt.Errorf("parse sse event: %w", err)
		}
		if onEvent == nil {
			return nil
		}
		return onEvent(event)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if after, ok := strings.CutPrefix(line, "data:"); ok {
			dataLines = append(dataLines, strings.TrimSpace(after))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read sse: %w", err)
	}
	return flush()
}
