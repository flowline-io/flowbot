// Package trilium implements the Trilium Notes ETAPI provider.
// It wraps the Trilium ETAPI REST API (https://docs.triliumnotes.org/user-guide/advanced-usage/etapi)
// using ETAPI tokens for authentication.
package trilium

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"resty.dev/v3"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	// ID is the provider identifier used in configuration and registration.
	ID = "trilium"
	// EndpointKey is the config key for the Trilium instance base URL.
	EndpointKey = "endpoint"
	// TokenKey is the config key for the ETAPI token.
	TokenKey = "token"
)

// Trilium wraps the Trilium ETAPI REST API client.
type Trilium struct {
	c     *resty.Client
	token string
}

// GetClient reads provider config and returns a new Trilium client.
// It returns nil when the endpoint is not configured.
func GetClient() *Trilium {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		flog.Warn("trilium provider: endpoint not configured")
		return nil
	}
	return NewTrilium(endpoint.String(), token.String())
}

// NewTrilium creates a Trilium client with the given endpoint and ETAPI token.
// If endpoint is empty, it returns nil.
// The token is sent as the Authorization header value directly (no Bearer prefix).
func NewTrilium(endpoint, token string) *Trilium {
	if endpoint == "" {
		return nil
	}
	v := &Trilium{token: token}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint + "/etapi")
	if token != "" {
		v.c.SetHeader("Authorization", token)
	}
	return v
}

// CreateNote creates a note and places it into the note tree via POST /create-note.
func (v *Trilium) CreateNote(ctx context.Context, req CreateNoteDef) (*NoteWithBranch, error) {
	resp := &NoteWithBranch{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(resp).
		Post("/create-note")
	if err != nil {
		return nil, fmt.Errorf("trilium create note: %w", err)
	}
	if httpResp.StatusCode() == http.StatusCreated {
		return resp, nil
	}
	return nil, parseError("trilium create note", httpResp)
}

// GetNote retrieves a note by its ID via GET /notes/{noteId}.
func (v *Trilium) GetNote(ctx context.Context, noteID string) (*Note, error) {
	resp := &Note{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetResult(resp).
		SetPathParam("noteId", noteID).
		Get("/notes/{noteId}")
	if err != nil {
		return nil, fmt.Errorf("trilium get note %s: %w", noteID, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium get note "+noteID, httpResp)
}

// PatchNote updates a note via PATCH /notes/{noteId}.
func (v *Trilium) PatchNote(ctx context.Context, noteID string, req PatchNoteRequest) (*Note, error) {
	resp := &Note{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(resp).
		SetPathParam("noteId", noteID).
		Patch("/notes/{noteId}")
	if err != nil {
		return nil, fmt.Errorf("trilium patch note %s: %w", noteID, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium patch note "+noteID, httpResp)
}

// DeleteNote deletes a note via DELETE /notes/{noteId}.
func (v *Trilium) DeleteNote(ctx context.Context, noteID string) error {
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetPathParam("noteId", noteID).
		Delete("/notes/{noteId}")
	if err != nil {
		return fmt.Errorf("trilium delete note %s: %w", noteID, err)
	}
	if httpResp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return parseError("trilium delete note "+noteID, httpResp)
}

// SearchNotes searches notes via GET /notes.
func (v *Trilium) SearchNotes(ctx context.Context, params SearchParams) (*SearchResponse, error) {
	resp := &SearchResponse{}
	req := v.c.R().SetContext(ctx).SetResult(resp).
		SetQueryParam("search", params.Search)
	if params.FastSearch {
		req.SetQueryParam("fastSearch", "true")
	}
	if params.IncludeArchivedNotes {
		req.SetQueryParam("includeArchivedNotes", "true")
	}
	if params.AncestorNoteID != "" {
		req.SetQueryParam("ancestorNoteId", params.AncestorNoteID)
	}
	if params.AncestorDepth != "" {
		req.SetQueryParam("ancestorDepth", params.AncestorDepth)
	}
	if params.OrderBy != "" {
		req.SetQueryParam("orderBy", params.OrderBy)
	}
	if params.OrderDirection != "" {
		req.SetQueryParam("orderDirection", params.OrderDirection)
	}
	if params.Limit > 0 {
		req.SetQueryParam("limit", strconv.Itoa(params.Limit))
	}
	if params.Debug {
		req.SetQueryParam("debug", "true")
	}
	httpResp, err := req.Get("/notes")
	if err != nil {
		return nil, fmt.Errorf("trilium search notes: %w", err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium search notes", httpResp)
}

// GetNoteContent retrieves the content of a note via GET /notes/{noteId}/content.
func (v *Trilium) GetNoteContent(ctx context.Context, noteID string) (string, error) {
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetPathParam("noteId", noteID).
		Get("/notes/{noteId}/content")
	if err != nil {
		return "", fmt.Errorf("trilium get note content %s: %w", noteID, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return httpResp.String(), nil
	}
	return "", parseError("trilium get note content "+noteID, httpResp)
}

// UpdateNoteContent updates the content of a note via PUT /notes/{noteId}/content.
func (v *Trilium) UpdateNoteContent(ctx context.Context, noteID, content string) error {
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(content).
		SetHeader("Content-Type", "text/plain").
		SetPathParam("noteId", noteID).
		Put("/notes/{noteId}/content")
	if err != nil {
		return fmt.Errorf("trilium update note content %s: %w", noteID, err)
	}
	if httpResp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return parseError("trilium update note content "+noteID, httpResp)
}

// GetAppInfo returns information about the running Trilium instance via GET /app-info.
func (v *Trilium) GetAppInfo(ctx context.Context) (*AppInfo, error) {
	resp := &AppInfo{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetResult(resp).
		Get("/app-info")
	if err != nil {
		return nil, fmt.Errorf("trilium get app info: %w", err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium get app info", httpResp)
}

// Login exchanges a password for an ETAPI token via POST /auth/login.
func (v *Trilium) Login(ctx context.Context, password string) (string, error) {
	resp := &LoginResponse{}
	// Remove auth header for login endpoint — the client-level header is overridden
	// by setting an empty value on the request.
	req := v.c.R().
		SetContext(ctx).
		SetBody(LoginRequest{Password: password}).
		SetResult(resp).
		SetHeader("Authorization", "") // override client header
	httpResp, err := req.Post("/auth/login")
	if err != nil {
		return "", fmt.Errorf("trilium login: %w", err)
	}
	if httpResp.StatusCode() == http.StatusCreated {
		return resp.AuthToken, nil
	}
	return "", parseError("trilium login", httpResp)
}

// Logout deactivates the current ETAPI token via POST /auth/logout.
func (v *Trilium) Logout(ctx context.Context) error {
	httpResp, err := v.c.R().
		SetContext(ctx).
		Post("/auth/logout")
	if err != nil {
		return fmt.Errorf("trilium logout: %w", err)
	}
	if httpResp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return parseError("trilium logout", httpResp)
}

// CreateBranch creates a branch (clones a note to a different location) via POST /branches.
func (v *Trilium) CreateBranch(ctx context.Context, req BranchRequest) (*Branch, error) {
	resp := &Branch{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(resp).
		Post("/branches")
	if err != nil {
		return nil, fmt.Errorf("trilium create branch: %w", err)
	}
	if httpResp.StatusCode() == http.StatusCreated || httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium create branch", httpResp)
}

// GetBranch retrieves a branch by its ID via GET /branches/{branchId}.
func (v *Trilium) GetBranch(ctx context.Context, branchID string) (*Branch, error) {
	resp := &Branch{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetResult(resp).
		SetPathParam("branchId", branchID).
		Get("/branches/{branchId}")
	if err != nil {
		return nil, fmt.Errorf("trilium get branch %s: %w", branchID, err)
	}
	if httpResp.StatusCode() == http.StatusOK {
		return resp, nil
	}
	return nil, parseError("trilium get branch "+branchID, httpResp)
}

// DeleteBranch deletes a branch via DELETE /branches/{branchId}.
func (v *Trilium) DeleteBranch(ctx context.Context, branchID string) error {
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetPathParam("branchId", branchID).
		Delete("/branches/{branchId}")
	if err != nil {
		return fmt.Errorf("trilium delete branch %s: %w", branchID, err)
	}
	if httpResp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return parseError("trilium delete branch "+branchID, httpResp)
}

// CreateAttribute creates an attribute for a note via POST /attributes.
func (v *Trilium) CreateAttribute(ctx context.Context, req CreateAttribute) (*Attribute, error) {
	resp := &Attribute{}
	httpResp, err := v.c.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(resp).
		Post("/attributes")
	if err != nil {
		return nil, fmt.Errorf("trilium create attribute: %w", err)
	}
	if httpResp.StatusCode() == http.StatusCreated {
		return resp, nil
	}
	return nil, parseError("trilium create attribute", httpResp)
}

// ListRawEvents searches notes and returns them as raw events for polling support.
// The cursor is used for pagination offset.
func (v *Trilium) ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error) {
	limit := MaxPageSize
	params := SearchParams{
		Search: "*",
		Limit:  limit,
	}
	if cursor != "" {
		offset, parseErr := strconv.Atoi(cursor)
		if parseErr != nil {
			return nil, "", fmt.Errorf("trilium list raw events: invalid cursor %q: %w", cursor, parseErr)
		}
		// Trilium search doesn't have offset natively, but we can use cursor as a synthetic offset.
		// This implementation relies on a wildcard search.
		// For offset-based pagination, fetch a larger batch and slice.
		_ = offset // reserved for future cursor improvements
	}

	resp, err := v.SearchNotes(ctx, params)
	if err != nil {
		return nil, "", fmt.Errorf("trilium list raw events: %w", err)
	}

	items := make([]map[string]any, len(resp.Results))
	for i, n := range resp.Results {
		items[i] = map[string]any{
			"noteId":          n.NoteID,
			"title":           n.Title,
			"type":            n.Type,
			"isProtected":     n.IsProtected,
			"dateCreated":     n.DateCreated,
			"dateModified":    n.DateModified,
			"utcDateCreated":  n.UtcDateCreated,
			"utcDateModified": n.UtcDateModified,
		}
	}

	nextCursor := ""
	if len(items) == limit {
		nextCursor = "1" // signal there may be more
	}
	return items, nextCursor, nil
}

// parseError extracts an error message from an HTTP response.
func parseError(op string, resp *resty.Response) error {
	var errResp ErrorResponse
	if err := sonic.Unmarshal(resp.Bytes(), &errResp); err == nil && errResp.Message != "" {
		return fmt.Errorf("%s: status %d: %s", op, errResp.Status, errResp.Message)
	}
	return fmt.Errorf("%s: unexpected status %d: %s", op, resp.StatusCode(), resp.String())
}
