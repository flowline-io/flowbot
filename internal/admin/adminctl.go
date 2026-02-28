// Package adminctl implements the Admin panel API controller, auth middleware,
// and go-app PWA handler registration.
//
// This package is intentionally standalone — it only depends on lightweight
// libraries (Fiber, go-app, flog, protocol, admin types) so it can be imported
// by both the main server (internal/server) and the PWA server (cmd/app).
//
// The backend uses mock in-memory data storage for demonstration purposes.
// In production, this should be replaced with database storage.
package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	slackProvider "github.com/flowline-io/flowbot/pkg/providers/slack"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Options holds configurable parameters for the admin controller.
type Options struct {
	// SlackClientID is the Slack OAuth application client ID.
	SlackClientID string
	// SlackClientSecret is the Slack OAuth application client secret.
	SlackClientSecret string
	// OAuthStore is an optional callback invoked after a successful Slack OAuth
	// exchange. It receives the user ID and the access token + extra data so
	// the caller can persist them (e.g. to the oauth table). When nil the
	// token is only kept in memory.
	OAuthStore func(uid, accessToken string, extra []byte) error
}

// tokenTTL is the session token lifetime.
const tokenTTL = 24 * time.Hour

// stateTTL is the OAuth state parameter lifetime.
const stateTTL = 10 * time.Minute

// codeExchangeTTL is the one-time code lifetime.
const codeExchangeTTL = 2 * time.Minute

// tokenEntry wraps user info with an expiration timestamp.
type tokenEntry struct {
	User      admin.UserInfo
	ExpiresAt time.Time
}

// timedValue is a generic value with an expiration timestamp.
type timedValue struct {
	Value     string
	ExpiresAt time.Time
}

// ---------------------------------------------------------------------------
// AdminController
// ---------------------------------------------------------------------------

// AdminController is the admin panel controller, holding mock data storage and business logic.
type AdminController struct {
	mu         sync.RWMutex
	containers []admin.Container
	nextID     atomic.Int64
	settings   admin.Settings
	tokens     sync.Map // token(string) -> tokenEntry
	states     sync.Map // state(string) -> timedValue (CSRF nonce)
	codes      sync.Map // code(string)  -> timedValue (one-time exchange code -> session token)
	opts       Options
}

// NewAdminController creates an AdminController instance with the given options.
func NewAdminController(opts Options) *AdminController {
	ctl := &AdminController{
		opts: opts,
		settings: admin.Settings{
			SiteName:       "Flowbot",
			LogoURL:        "",
			SEODescription: "Flowbot - Intelligent Chatbot Platform",
			MaxUploadSize:  10 * 1024 * 1024, // 10MB
		},
	}
	ctl.nextID.Store(1)
	ctl.initMockData()
	go ctl.cleanupLoop()
	return ctl
}

// cleanupLoop periodically removes expired tokens, states, and codes.
func (ac *AdminController) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		ac.tokens.Range(func(key, value any) bool {
			if entry, ok := value.(tokenEntry); ok && now.After(entry.ExpiresAt) {
				ac.tokens.Delete(key)
			}
			return true
		})
		ac.states.Range(func(key, value any) bool {
			if entry, ok := value.(timedValue); ok && now.After(entry.ExpiresAt) {
				ac.states.Delete(key)
			}
			return true
		})
		ac.codes.Range(func(key, value any) bool {
			if entry, ok := value.(timedValue); ok && now.After(entry.ExpiresAt) {
				ac.codes.Delete(key)
			}
			return true
		})
	}
}

// initMockData initializes example container data.
func (ac *AdminController) initMockData() {
	now := time.Now()
	ac.containers = []admin.Container{
		{ID: 1, Name: "nginx-proxy", Status: admin.ContainerRunning, CreatedAt: now.Add(-72 * time.Hour)},
		{ID: 2, Name: "redis-cache", Status: admin.ContainerRunning, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: 3, Name: "postgres-db", Status: admin.ContainerStopped, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: 4, Name: "app-backend", Status: admin.ContainerRunning, CreatedAt: now.Add(-12 * time.Hour)},
		{ID: 5, Name: "monitoring", Status: admin.ContainerPaused, CreatedAt: now.Add(-6 * time.Hour)},
		{ID: 6, Name: "rabbitmq", Status: admin.ContainerRunning, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 7, Name: "elasticsearch", Status: admin.ContainerStopped, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 8, Name: "minio-storage", Status: admin.ContainerRunning, CreatedAt: now.Add(-1 * time.Hour)},
	}
	ac.nextID.Store(9)
}

// ---------------------------------------------------------------------------
// go-app PWA Handler & Route Registration
// ---------------------------------------------------------------------------

// NewAppHandler creates the go-app HTTP Handler that serves Wasm and
// static assets. Tailwind CSS and DaisyUI are loaded via CDN.
// apiBaseURL is passed to the Wasm client via Handler.Env so the frontend
// knows which backend API endpoint to call (e.g. "http://127.0.0.1:8060/service/admin").
func NewAppHandler(apiBaseURL string) http.Handler {
	h := &app.Handler{
		Name:        "Flowbot Admin",
		ShortName:   "FBAdmin",
		Description: "Flowbot Admin Panel",
		Author:      "Flowline",
		Styles: []string{
			"https://cdn.jsdelivr.net/npm/daisyui@4/dist/full.min.css",
		},
		RawHeaders: []string{
			`<script src="https://cdn.tailwindcss.com"></script>`,
		},
	}
	if apiBaseURL != "" {
		h.Env = map[string]string{
			"API_BASE_URL": apiBaseURL,
		}
	}
	return h
}

// HandleAPIRoutes registers Admin API endpoints (/service/admin/*) on the given Fiber app.
// Used by the main server (cmd/main.go) which provides the backend API.
func HandleAPIRoutes(a *fiber.App, ac *AdminController) {
	adminAPI := a.Group("/service/admin")

	// Auth endpoints (no token required)
	adminAPI.Get("/auth/slack/url", ac.getSlackOAuthURL)
	adminAPI.Get("/auth/slack/callback", ac.handleSlackCallback)
	adminAPI.Post("/auth/exchange", ac.exchangeCode)
	adminAPI.Post("/auth/dev-login", ac.devLogin)

	// Authenticated API endpoints
	adminAPI.Get("/auth/me", ac.adminAuth(ac.getCurrentUser))
	adminAPI.Get("/settings", ac.adminAuth(ac.getSettings))
	adminAPI.Put("/settings", ac.adminAuth(ac.updateSettings))
	adminAPI.Get("/containers", ac.adminAuth(ac.listContainers))
	adminAPI.Post("/containers", ac.adminAuth(ac.createContainer))
	adminAPI.Post("/containers/batch-delete", ac.adminAuth(ac.batchDeleteContainers))
	adminAPI.Get("/containers/:id", ac.adminAuth(ac.getContainer))
	adminAPI.Put("/containers/:id", ac.adminAuth(ac.updateContainer))
	adminAPI.Delete("/containers/:id", ac.adminAuth(ac.deleteContainer))

	log.Println("admin API routes registered")
}

// HandlePageRoutes registers go-app PWA static resource routes on the given Fiber app.
// Used by the PWA server (cmd/app) which serves the frontend.
// apiBaseURL is forwarded to the Wasm client via environment variables.
//
// A catch-all fallback is registered so the go-app Handler can serve all
// generated assets (app.css, app.js, app-worker.js, wasm_exec.js, manifest,
// web/* resources, etc.) without having to enumerate every path explicitly.
func HandlePageRoutes(a *fiber.App, apiBaseURL string) {
	appHandler := NewAppHandler(apiBaseURL)
	httpHandler := adaptor.HTTPHandler(appHandler)

	// Catch-all: let go-app handle every request (SPA pages + static assets).
	a.Use(httpHandler)

	log.Println("admin page routes registered")
}

// ---------------------------------------------------------------------------
// Auth middleware
// ---------------------------------------------------------------------------

// adminAuth is the Admin API authentication middleware.
func (ac *AdminController) adminAuth(handler fiber.Handler) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		authHeader := ctx.Get("Authorization")
		if authHeader == "" {
			return protocol.ErrNotAuthorized.New("missing Authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return protocol.ErrNotAuthorized.New("invalid Authorization format")
		}

		token := parts[1]
		v, ok := ac.tokens.Load(token)
		if !ok {
			return protocol.ErrNotAuthorized.New("token invalid or expired")
		}
		entry, ok := v.(tokenEntry)
		if !ok || time.Now().After(entry.ExpiresAt) {
			ac.tokens.Delete(token)
			return protocol.ErrNotAuthorized.New("token expired")
		}

		ctx.Locals("admin_token", token)
		return handler(ctx)
	}
}

// getTokenUser retrieves user info from a validated request.
func (ac *AdminController) getTokenUser(ctx fiber.Ctx) *admin.UserInfo {
	token, _ := ctx.Locals("admin_token").(string)
	if v, ok := ac.tokens.Load(token); ok {
		if entry, ok := v.(tokenEntry); ok {
			return &entry.User
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Auth API
// ---------------------------------------------------------------------------

// getSlackOAuthURL generates the Slack OAuth authorization URL.
func (ac *AdminController) getSlackOAuthURL(ctx fiber.Ctx) error {
	clientID := ac.opts.SlackClientID
	if clientID == "" {
		return protocol.ErrBadParam.New("Slack client ID is not configured")
	}

	redirectURI := ac.buildRedirectURI(ctx)
	state := ac.generateState()

	// Reuse the Slack provider to construct the authorize URL.
	provider := slackProvider.NewSlack(clientID, "", redirectURI, "")
	provider.SetState(state)

	return ctx.JSON(protocol.NewSuccessResponse(admin.SlackOAuthURLResponse{
		URL: provider.GetAuthorizeURL(),
	}))
}

// generateState creates a cryptographically random state string and stores it
// with a TTL for later validation.
func (ac *AdminController) generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)
	ac.states.Store(state, timedValue{
		Value:     state,
		ExpiresAt: time.Now().Add(stateTTL),
	})
	return state
}

// validateState checks that the state parameter returned by Slack matches a
// pending state and has not expired. The state is consumed (deleted) on success.
func (ac *AdminController) validateState(state string) bool {
	v, ok := ac.states.LoadAndDelete(state)
	if !ok {
		return false
	}
	entry, ok := v.(timedValue)
	if !ok {
		return false
	}
	return time.Now().Before(entry.ExpiresAt)
}

// buildRedirectURI constructs the OAuth callback URL based on the current request.
func (ac *AdminController) buildRedirectURI(ctx fiber.Ctx) string {
	scheme := "https"
	if ctx.Protocol() == "http" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/service/admin/auth/slack/callback", scheme, ctx.Hostname())
}

// handleSlackCallback handles the Slack OAuth callback.
func (ac *AdminController) handleSlackCallback(ctx fiber.Ctx) error {
	// Validate the CSRF state parameter first.
	state := ctx.Query("state")
	if state == "" || !ac.validateState(state) {
		log.Printf("slack oauth callback: invalid or expired state")
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid OAuth state — please try again"))
	}

	code := ctx.Query("code")
	if code == "" {
		errMsg := ctx.Query("error", "missing code parameter")
		log.Printf("slack oauth callback error: %s", errMsg)
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	clientID := ac.opts.SlackClientID
	clientSecret := ac.opts.SlackClientSecret
	if clientID == "" || clientSecret == "" {
		log.Printf("slack oauth: client ID or secret not configured")
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Slack OAuth is not configured"))
	}

	redirectURI := ac.buildRedirectURI(ctx)

	// Step 1: Exchange the authorization code for an access token via the Slack provider.
	provider := slackProvider.NewSlack(clientID, clientSecret, redirectURI, "")
	kv, err := provider.GetAccessToken(ctx)
	if err != nil {
		log.Printf("slack oauth token exchange failed: %v", err)
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Slack authentication failed"))
	}

	accessToken, _ := kv["token"].(string)

	// Step 2: Use the same provider instance to fetch the user's identity
	// (accessToken is already set internally after GetAccessToken).
	identity, err := provider.GetIdentity()
	if err != nil {
		log.Printf("slack oauth identity fetch failed: %v", err)
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Failed to retrieve Slack user info"))
	}

	user := admin.UserInfo{
		UID:      identity.User.ID,
		Name:     identity.User.Name,
		Avatar:   identity.User.Image48,
		Platform: "slack",
	}

	// Step 3: Persist the OAuth token to the database (if a store callback is provided).
	if ac.opts.OAuthStore != nil {
		extra, _ := kv["extra"].([]byte)
		if err := ac.opts.OAuthStore(user.UID, accessToken, extra); err != nil {
			log.Printf("slack oauth store failed: %v", err)
			// Non-fatal: the user can still log in even if persistence fails.
		}
	}

	log.Printf("slack oauth login successful: uid=%s name=%s", user.UID, user.Name)

	// Step 4: Create a session token and wrap it in a one-time exchange code
	// so the real token never appears in the URL / browser history.
	sessionToken := ac.createToken(user)
	exchangeCode := ac.createExchangeCode(sessionToken)
	return ctx.Redirect().To("/admin/login?code=" + url.QueryEscape(exchangeCode))
}

// exchangeCode handles POST /auth/exchange — swaps a one-time code for a
// session token. The code is consumed (deleted) on first use.
func (ac *AdminController) exchangeCode(ctx fiber.Ctx) error {
	var req admin.CodeExchangeRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}
	if req.Code == "" {
		return protocol.ErrBadParam.New("missing code")
	}

	v, ok := ac.codes.LoadAndDelete(req.Code)
	if !ok {
		return protocol.ErrNotAuthorized.New("invalid or expired code")
	}
	entry, ok := v.(timedValue)
	if !ok || time.Now().After(entry.ExpiresAt) {
		return protocol.ErrNotAuthorized.New("code expired")
	}

	return ctx.JSON(protocol.NewSuccessResponse(admin.TokenResponse{
		Token: entry.Value,
	}))
}

// createExchangeCode generates a short-lived one-time code that maps to a
// session token. The caller redirects with this code instead of the real token.
func (ac *AdminController) createExchangeCode(sessionToken string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	code := hex.EncodeToString(b)
	ac.codes.Store(code, timedValue{
		Value:     sessionToken,
		ExpiresAt: time.Now().Add(codeExchangeTTL),
	})
	return code
}

// devLogin performs a quick dev-mode login (no Slack OAuth required).
func (ac *AdminController) devLogin(ctx fiber.Ctx) error {
	user := admin.UserInfo{
		UID:      "dev-admin",
		Name:     "Admin",
		Avatar:   "",
		Platform: "dev",
	}

	token := ac.createToken(user)

	return ctx.JSON(protocol.NewSuccessResponse(admin.TokenResponse{
		Token: token,
	}))
}

// createToken generates an access token with TTL and stores the user info mapping.
func (ac *AdminController) createToken(user admin.UserInfo) string {
	token := uuid.New().String()
	ac.tokens.Store(token, tokenEntry{
		User:      user,
		ExpiresAt: time.Now().Add(tokenTTL),
	})
	return token
}

// getCurrentUser retrieves the current logged-in user's info.
func (ac *AdminController) getCurrentUser(ctx fiber.Ctx) error {
	user := ac.getTokenUser(ctx)
	if user == nil {
		return protocol.ErrNotAuthorized.New("user info not found")
	}
	return ctx.JSON(protocol.NewSuccessResponse(user))
}

// ---------------------------------------------------------------------------
// System Settings API
// ---------------------------------------------------------------------------

// getSettings retrieves system settings.
func (ac *AdminController) getSettings(ctx fiber.Ctx) error {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ctx.JSON(protocol.NewSuccessResponse(ac.settings))
}

// updateSettings updates the system settings.
func (ac *AdminController) updateSettings(ctx fiber.Ctx) error {
	var req admin.Settings
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	ac.mu.Lock()
	ac.settings = req
	ac.mu.Unlock()

	log.Printf("admin settings updated: %+v", req)
	return ctx.JSON(protocol.NewSuccessResponse(ac.settings))
}

// ---------------------------------------------------------------------------
// Container Management API (Mock CRUD)
// ---------------------------------------------------------------------------

// listContainers returns a paginated, searchable, sortable container list.
func (ac *AdminController) listContainers(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("page_size", "10"))
	search := ctx.Query("search")
	sortBy := ctx.Query("sort_by")
	sortDesc := ctx.Query("sort_desc") == "true"

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	ac.mu.RLock()
	all := make([]admin.Container, len(ac.containers))
	copy(all, ac.containers)
	ac.mu.RUnlock()

	// Search filter
	if search != "" {
		filtered := make([]admin.Container, 0)
		searchLower := strings.ToLower(search)
		for _, c := range all {
			if strings.Contains(strings.ToLower(c.Name), searchLower) {
				filtered = append(filtered, c)
			}
		}
		all = filtered
	}

	// Sort
	if sortBy != "" {
		sort.Slice(all, func(i, j int) bool {
			less := false
			switch sortBy {
			case "id":
				less = all[i].ID < all[j].ID
			case "name":
				less = all[i].Name < all[j].Name
			case "status":
				less = string(all[i].Status) < string(all[j].Status)
			case "created_at":
				less = all[i].CreatedAt.Before(all[j].CreatedAt)
			default:
				less = all[i].ID < all[j].ID
			}
			if sortDesc {
				return !less
			}
			return less
		})
	}

	// Paginate
	total := int64(len(all))
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	items := all[start:end]

	return ctx.JSON(protocol.NewSuccessResponse(admin.ListResponse[admin.Container]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}))
}

// createContainer creates a new container.
func (ac *AdminController) createContainer(ctx fiber.Ctx) error {
	var req admin.ContainerCreateRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if req.Name == "" {
		return protocol.ErrBadParam.New("container name cannot be empty")
	}

	newContainer := admin.Container{
		ID:        ac.nextID.Add(1) - 1,
		Name:      req.Name,
		Status:    req.Status,
		CreatedAt: time.Now(),
	}

	ac.mu.Lock()
	ac.containers = append(ac.containers, newContainer)
	ac.mu.Unlock()

	log.Printf("admin container created: %+v", newContainer)
	return ctx.JSON(protocol.NewSuccessResponse(newContainer))
}

// getContainer retrieves a single container by ID.
func (ac *AdminController) getContainer(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, c := range ac.containers {
		if c.ID == id {
			return ctx.JSON(protocol.NewSuccessResponse(c))
		}
	}

	return protocol.ErrNotFound.New("container not found")
}

// updateContainer updates an existing container.
func (ac *AdminController) updateContainer(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	var req admin.ContainerUpdateRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	for i, c := range ac.containers {
		if c.ID == id {
			if req.Name != "" {
				ac.containers[i].Name = req.Name
			}
			if req.Status != "" {
				ac.containers[i].Status = req.Status
			}
			log.Printf("admin container updated: %+v", ac.containers[i])
			return ctx.JSON(protocol.NewSuccessResponse(ac.containers[i]))
		}
	}

	return protocol.ErrNotFound.New("container not found")
}

// deleteContainer removes a container by ID.
func (ac *AdminController) deleteContainer(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	for i, c := range ac.containers {
		if c.ID == id {
			ac.containers = append(ac.containers[:i], ac.containers[i+1:]...)
			log.Printf("admin container deleted: id=%d", id)
			return ctx.JSON(protocol.NewSuccessResponse(nil))
		}
	}

	return protocol.ErrNotFound.New("container not found")
}

// batchDeleteContainers removes multiple containers by their IDs.
func (ac *AdminController) batchDeleteContainers(ctx fiber.Ctx) error {
	var req admin.BatchDeleteRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if len(req.IDs) == 0 {
		return protocol.ErrBadParam.New("ID list cannot be empty")
	}

	deleteSet := make(map[int64]bool, len(req.IDs))
	for _, id := range req.IDs {
		deleteSet[id] = true
	}

	ac.mu.Lock()
	filtered := make([]admin.Container, 0, len(ac.containers))
	deleted := 0
	for _, c := range ac.containers {
		if deleteSet[c.ID] {
			deleted++
		} else {
			filtered = append(filtered, c)
		}
	}
	ac.containers = filtered
	ac.mu.Unlock()

	log.Printf("admin containers batch deleted: %d items", deleted)
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}
