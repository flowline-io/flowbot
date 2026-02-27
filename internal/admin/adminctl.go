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
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	tokens     sync.Map // token(string) -> admin.UserInfo
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
	return ctl
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
func HandlePageRoutes(a *fiber.App, apiBaseURL string) {
	appHandler := NewAppHandler(apiBaseURL)
	httpHandler := adaptor.HTTPHandler(appHandler)

	// Admin page entry (all /admin/* paths return the SPA HTML)
	a.Get("/admin", httpHandler)
	a.Get("/admin/*", httpHandler)

	// go-app runtime assets
	a.Get("/web/*", httpHandler)
	a.Get("/app.js", httpHandler)
	a.Get("/app-worker.js", httpHandler)
	a.Get("/manifest.webmanifest", httpHandler)
	a.Get("/wasm_exec.js", httpHandler)

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
		if _, ok := ac.tokens.Load(token); !ok {
			return protocol.ErrNotAuthorized.New("token invalid or expired")
		}

		ctx.Locals("admin_token", token)
		return handler(ctx)
	}
}

// getTokenUser retrieves user info from a validated request.
func (ac *AdminController) getTokenUser(ctx fiber.Ctx) *admin.UserInfo {
	token, _ := ctx.Locals("admin_token").(string)
	if v, ok := ac.tokens.Load(token); ok {
		user := v.(admin.UserInfo)
		return &user
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
		clientID = "YOUR_SLACK_CLIENT_ID"
	}

	scheme := "http"
	host := ctx.Hostname()
	redirectURI := fmt.Sprintf("%s://%s/service/admin/auth/slack/callback", scheme, host)

	oauthURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&user_scope=identity.basic,identity.avatar&redirect_uri=%s",
		clientID,
		redirectURI,
	)

	return ctx.JSON(protocol.NewSuccessResponse(admin.SlackOAuthURLResponse{
		URL: oauthURL,
	}))
}

// handleSlackCallback handles the Slack OAuth callback.
func (ac *AdminController) handleSlackCallback(ctx fiber.Ctx) error {
	code := ctx.Query("code")
	if code == "" {
		return protocol.ErrBadParam.New("missing code parameter")
	}

	// TODO: In production, exchange the code for user info via Slack API:
	//   1. POST https://slack.com/api/oauth.v2.access to get access_token
	//   2. GET https://slack.com/api/users.identity to get user info
	// Currently a mock implementation that generates a test user.

	log.Printf("slack oauth callback received, code=%s", code)

	user := admin.UserInfo{
		UID:      "slack-user-" + code[:8],
		Name:     "Slack User",
		Avatar:   "",
		Platform: "slack",
	}

	token := ac.createToken(user)
	return ctx.Redirect().To(fmt.Sprintf("/admin/login?token=%s", token))
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

// createToken generates an access token and stores the user info mapping.
func (ac *AdminController) createToken(user admin.UserInfo) string {
	token := uuid.New().String()
	ac.tokens.Store(token, user)
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
