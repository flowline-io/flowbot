// Package adminctl implements the Admin panel API controller, auth middleware,
// and go-app PWA handler registration.
//
// This package is intentionally standalone — it only depends on lightweight
// libraries (Fiber, go-app, flog, protocol, admin types) so it can be imported
// by both the main server (internal/server) and the PWA server (cmd/app).
package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flowline-io/flowbot/internal/store/dao"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	slackProvider "github.com/flowline-io/flowbot/pkg/providers/slack"
	"github.com/flowline-io/flowbot/pkg/types/admin"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/pkg/utils"
	versionPkg "github.com/flowline-io/flowbot/version"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

var themeCSS = `/* Flowbot Admin - Refined Color Theme */
:root,[data-theme="light"]{color-scheme:light;--color-base-100:#ffffff;--color-base-200:#f8fafc;--color-base-300:#f1f5f9;--color-base-content:#1e293b;--color-primary:#4f46e5;--color-primary-content:#ffffff;--color-primary-focus:#4338ca;--color-primary-active:#3730a3;--color-secondary:#64748b;--color-secondary-content:#ffffff;--color-accent:#14b8a6;--color-accent-content:#ffffff;--color-neutral:#334155;--color-neutral-content:#ffffff;--color-base:#ffffff;--color-base-emphasis:#0f172a;--color-base-highlight:#f8fafc;--color-info:#0ea5e9;--color-info-content:#ffffff;--color-success:#10b981;--color-success-content:#ffffff;--color-warning:#f59e0b;--color-warning-content:#ffffff;--color-error:#ef4444;--color-error-content:#ffffff;--rounded-box:1rem;--rounded-btn:0.625rem;--rounded-badge:1.9rem;--animation-btn:0.2s;--animation-input:0.2s;--btn-focus-scale:0.98;--border-btn:1px;--tab-border:1px;--tab-radius:0.5rem}
[data-theme="dark"]{color-scheme:dark;--color-base-100:#0f172a;--color-base-200:#1e293b;--color-base-300:#334155;--color-base-content:#f1f5f9;--color-primary:#818cf8;--color-primary-content:#1e1b4b;--color-primary-focus:#a5b4fc;--color-primary-active:#6366f1;--color-secondary:#94a3b8;--color-secondary-content:#1e293b;--color-accent:#2dd4bf;--color-accent-content:#042f2e;--color-neutral:#475569;--color-neutral-content:#f8fafc;--color-base:#0f172a;--color-base-emphasis:#f8fafc;--color-base-highlight:#1e293b;--color-info:#38bdf8;--color-info-content:#0c4a6e;--color-success:#34d399;--color-success-content:#022c22;--color-warning:#fbbf24;--color-warning-content:#451a03;--color-error:#f87171;--color-error-content:#450a0a}
*{font-family:'Inter',ui-sans-serif,system-ui,sans-serif}
::-webkit-scrollbar{width:8px;height:8px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--color-base-300);border-radius:4px}
::-webkit-scrollbar-thumb:hover{background:var(--color-neutral)}
::selection{background:var(--color-primary);color:var(--color-primary-content)}
:focus-visible{outline:2px solid var(--color-primary);outline-offset:2px}
body{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale;scroll-behavior:smooth}
.gradient-primary{background:linear-gradient(135deg,var(--color-primary) 0%,color-mix(in srgb,var(--color-primary) 70%,white) 100%)}
.gradient-dark{background:linear-gradient(135deg,var(--color-base-100) 0%,var(--color-base-200) 100%)}
.glass{background:rgba(255,255,255,0.1);backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px)}
[data-theme="dark"] .glass{background:rgba(15,23,42,0.7)}
.shadow-soft{box-shadow:0 2px 15px -3px rgba(0,0,0,0.07),0 10px 20px -2px rgba(0,0,0,0.04)}
.shadow-glow-primary{box-shadow:0 0 20px rgba(79,70,229,0.3)}
.shadow-glow-success{box-shadow:0 0 20px rgba(16,185,129,0.3)}
.text-gradient{background:linear-gradient(135deg,var(--color-primary) 0%,var(--color-accent) 100%);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.border-soft{border-color:var(--color-base-300)}
.badge-soft{backdrop-filter:blur(8px);background:color-mix(in srgb,var(--color-base-200) 80%,transparent)}
.btn-glow:hover{box-shadow:0 0 20px color-mix(in srgb,var(--color-primary) 40%,transparent)}
.card-lift{transition:transform 0.3s ease,box-shadow 0.3s ease}
.card-lift:hover{transform:translateY(-4px);box-shadow:0 20px 25px -5px rgba(0,0,0,0.1),0 8px 10px -6px rgba(0,0,0,0.1)}
@keyframes pulse-soft{0%,100%{opacity:1}50%{opacity:0.7}}
.animate-pulse-soft{animation:pulse-soft 2s cubic-bezier(0.4,0,0.6,1) infinite}
.stagger-item{opacity:0;transform:translateY(10px);animation:stagger-fade 0.4s ease forwards}
@keyframes stagger-fade{to{opacity:1;transform:translateY(0)}}
.stagger-item:nth-child(1){animation-delay:0.05s}.stagger-item:nth-child(2){animation-delay:0.1s}.stagger-item:nth-child(3){animation-delay:0.15s}.stagger-item:nth-child(4){animation-delay:0.2s}.stagger-item:nth-child(5){animation-delay:0.25s}.stagger-item:nth-child(6){animation-delay:0.3s}.stagger-item:nth-child(7){animation-delay:0.35s}.stagger-item:nth-child(8){animation-delay:0.4s}
@keyframes shimmer{0%{background-position:-200% 0}100%{background-position:200% 0}}
.skeleton-shimmer{background:linear-gradient(90deg,var(--color-base-200) 25%,var(--color-base-300) 50%,var(--color-base-200) 75%);background-size:200% 100%;animation:shimmer 1.5s infinite}`

// Options holds configurable parameters for the admin controller.
type Options struct {
	// SlackClientID is the Slack OAuth application client ID.
	SlackClientID string
	// SlackClientSecret is the Slack OAuth application client secret.
	SlackClientSecret string
	// DevMode enables dev-login endpoint and related features.
	DevMode bool
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

// AdminController is the admin panel controller, holding business logic.
type AdminController struct {
	mu        sync.RWMutex
	settings  admin.Settings
	tokens    sync.Map // token(string) -> tokenEntry
	states    sync.Map // state(string) -> timedValue (CSRF nonce)
	codes     sync.Map // code(string)  -> timedValue (one-time exchange code -> session token)
	opts      Options
	startTime time.Time // server start time for uptime calculation
}

// NewAdminController creates an AdminController instance with the given options.
func NewAdminController(opts Options) *AdminController {
	ctl := &AdminController{
		opts:      opts,
		startTime: time.Now(),
		settings: admin.Settings{
			SiteName:       "Flowbot",
			LogoURL:        "",
			SEODescription: "Flowbot - Intelligent Chatbot Platform",
			MaxUploadSize:  10 * 1024 * 1024, // 10MB
		},
	}
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

// ---------------------------------------------------------------------------
// go-app PWA Handler & Route Registration
// ---------------------------------------------------------------------------

// NewAppHandler creates the go-app HTTP Handler that serves Wasm and
// static assets. Tailwind CSS and DaisyUI are loaded via CDN.
func NewAppHandler(apiBaseURL string, devMode bool) http.Handler {
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
			`<link rel="preconnect" href="https://fonts.googleapis.com">`,
			`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>`,
			`<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">`,
			`<style>` + themeCSS + `</style>`,
		},
	}
	if apiBaseURL != "" || devMode {
		h.Env = map[string]string{
			"API_BASE_URL": apiBaseURL,
			"DEV_MODE":     utils.BoolToString(devMode),
		}
	} else if apiBaseURL != "" {
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

	adminAPI.Get("/auth/slack/url", ac.getSlackOAuthURL)
	adminAPI.Get("/auth/slack/callback", ac.handleSlackCallback)
	adminAPI.Post("/auth/exchange", ac.exchangeCode)

	if ac.opts.DevMode {
		adminAPI.Post("/auth/dev-login", ac.devLogin)
		flog.Info("admin dev-login endpoint enabled")
	}

	// Authenticated API endpoints
	adminAPI.Get("/auth/me", ac.adminAuth(ac.getCurrentUser))
	adminAPI.Get("/settings", ac.adminAuth(ac.getSettings))
	adminAPI.Put("/settings", ac.adminAuth(ac.updateSettings))
	adminAPI.Get("/dashboard/stats", ac.adminAuth(ac.getDashboardStats))
	adminAPI.Get("/containers", ac.adminAuth(ac.listContainers))
	adminAPI.Post("/containers", ac.adminAuth(ac.createContainer))
	adminAPI.Post("/containers/batch-delete", ac.adminAuth(ac.batchDeleteContainers))
	adminAPI.Get("/containers/:id", ac.adminAuth(ac.getContainer))
	adminAPI.Put("/containers/:id", ac.adminAuth(ac.updateContainer))
	adminAPI.Delete("/containers/:id", ac.adminAuth(ac.deleteContainer))

	// User management endpoints
	adminAPI.Get("/users", ac.adminAuth(ac.listUsers))
	adminAPI.Post("/users", ac.adminAuth(ac.createUser))
	adminAPI.Get("/users/:id", ac.adminAuth(ac.getUser))
	adminAPI.Put("/users/:id", ac.adminAuth(ac.updateUser))
	adminAPI.Delete("/users/:id", ac.adminAuth(ac.deleteUser))

	// Workflow management endpoints
	adminAPI.Get("/workflows", ac.adminAuth(ac.listWorkflows))
	adminAPI.Post("/workflows", ac.adminAuth(ac.createWorkflow))
	adminAPI.Get("/workflows/:id", ac.adminAuth(ac.getWorkflow))
	adminAPI.Delete("/workflows/:id", ac.adminAuth(ac.deleteWorkflow))
	adminAPI.Post("/workflows/:id/run", ac.adminAuth(ac.runWorkflow))

	// Bot management endpoints
	adminAPI.Get("/bots", ac.adminAuth(ac.listBots))
	adminAPI.Get("/bots/:name", ac.adminAuth(ac.getBot))
	adminAPI.Post("/bots/:name/enable", ac.adminAuth(ac.enableBot))
	adminAPI.Post("/bots/:name/disable", ac.adminAuth(ac.disableBot))

	// Log viewer endpoints
	adminAPI.Get("/logs", ac.adminAuth(ac.listLogs))
	adminAPI.Get("/logs/sources", ac.adminAuth(ac.getLogSources))

	flog.Info("admin API routes registered")
}

// HandlePageRoutes registers go-app PWA static resource routes on the given Fiber app.
// Used by the PWA server (cmd/app) which serves the frontend.
// apiBaseURL is forwarded to the Wasm client via environment variables.
//
// A catch-all fallback is registered so the go-app Handler can serve all
// generated assets (app.css, app.js, app-worker.js, wasm_exec.js, manifest,
// web/* resources, etc.) without having to enumerate every path explicitly.
func HandlePageRoutes(a *fiber.App, apiBaseURL string, devMode bool) {
	appHandler := NewAppHandler(apiBaseURL, devMode)
	httpHandler := adaptor.HTTPHandler(appHandler)

	// Catch-all: let go-app handle every request (SPA pages + static assets).
	a.Use(httpHandler)

	flog.Info("admin page routes registered")
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
		flog.Info("slack oauth callback: invalid or expired state")
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Invalid OAuth state — please try again"))
	}

	code := ctx.Query("code")
	if code == "" {
		errMsg := ctx.Query("error", "missing code parameter")
		flog.Info("slack oauth callback error: %s", errMsg)
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape(errMsg))
	}

	clientID := ac.opts.SlackClientID
	clientSecret := ac.opts.SlackClientSecret
	if clientID == "" || clientSecret == "" {
		flog.Info("slack oauth: client ID or secret not configured")
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Slack OAuth is not configured"))
	}

	redirectURI := ac.buildRedirectURI(ctx)

	// Step 1: Exchange the authorization code for an access token via the Slack provider.
	provider := slackProvider.NewSlack(clientID, clientSecret, redirectURI, "")
	kv, err := provider.GetAccessToken(ctx)
	if err != nil {
		flog.Info("slack oauth token exchange failed: %v", err)
		return ctx.Redirect().To("/admin/login?error=" + url.QueryEscape("Slack authentication failed"))
	}

	accessToken, _ := kv["token"].(string)

	// Step 2: Use the same provider instance to fetch the user's identity
	// (accessToken is already set internally after GetAccessToken).
	identity, err := provider.GetIdentity()
	if err != nil {
		flog.Info("slack oauth identity fetch failed: %v", err)
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
			flog.Info("slack oauth store failed: %v", err)
			// Non-fatal: the user can still log in even if persistence fails.
		}
	}

	flog.Info("slack oauth login successful: uid=%s name=%s", user.UID, user.Name)

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

	flog.Info("admin settings updated: %+v", req)
	return ctx.JSON(protocol.NewSuccessResponse(ac.settings))
}

// ---------------------------------------------------------------------------
// Container Management API
// ---------------------------------------------------------------------------

// listContainers returns an empty list as containers are not stored in database.
func (ac *AdminController) listContainers(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return ctx.JSON(protocol.NewSuccessResponse(admin.ListResponse[admin.Container]{
		Items:      []admin.Container{},
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}))
}

// createContainer is not implemented as containers are not stored in database.
func (ac *AdminController) createContainer(ctx fiber.Ctx) error {
	return protocol.ErrUnsupported.New("container management is not available")
}

// getContainer is not implemented as containers are not stored in database.
func (ac *AdminController) getContainer(ctx fiber.Ctx) error {
	return protocol.ErrNotFound.New("container not found")
}

// updateContainer is not implemented as containers are not stored in database.
func (ac *AdminController) updateContainer(ctx fiber.Ctx) error {
	return protocol.ErrNotFound.New("container not found")
}

// deleteContainer is not implemented as containers are not stored in database.
func (ac *AdminController) deleteContainer(ctx fiber.Ctx) error {
	return protocol.ErrNotFound.New("container not found")
}

// batchDeleteContainers is not implemented as containers are not stored in database.
func (ac *AdminController) batchDeleteContainers(ctx fiber.Ctx) error {
	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// getDashboardStats returns aggregated statistics for the admin dashboard.
func (ac *AdminController) getDashboardStats(ctx fiber.Ctx) error {
	// Get counts from database
	userCount, err := dao.User.WithContext(ctx.Context()).Count()
	if err != nil {
		flog.Error(fmt.Errorf("failed to get user count: %w", err))
		userCount = 0
	}

	botCount, err := dao.Bot.WithContext(ctx.Context()).Count()
	if err != nil {
		flog.Error(fmt.Errorf("failed to get bot count: %w", err))
		botCount = 0
	}

	workflowCount, err := dao.Workflow.WithContext(ctx.Context()).Count()
	if err != nil {
		flog.Error(fmt.Errorf("failed to get workflow count: %w", err))
		workflowCount = 0
	}

	// Runtime info
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(ac.startTime)

	stats := admin.DashboardStats{
		TotalContainers:   int(workflowCount),
		RunningContainers: int(botCount),
		StoppedContainers: int(userCount),
		PausedContainers:  0,
		ErrorContainers:   0,

		Uptime:      formatDuration(uptime),
		GoVersion:   runtime.Version(),
		SystemOS:    runtime.GOOS,
		SystemArch:  runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		NumRoutines: runtime.NumGoroutine(),
		MemoryUsage: memStats.HeapAlloc,
		MemoryTotal: memStats.TotalAlloc,
		Version:     versionPkg.Buildtags,

		RecentContainers: []admin.Container{},
		ActivityLog:      []admin.ActivityEntry{},
	}

	return ctx.JSON(protocol.NewSuccessResponse(stats))
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// ---------------------------------------------------------------------------
// User management API
// ---------------------------------------------------------------------------

func modelUserToAdminUser(u *model.User) admin.User {
	status := admin.UserActive
	if u.State == model.UserInactive {
		status = admin.UserInactive
	}

	return admin.User{
		ID:        u.ID,
		UID:       u.Flag,
		Name:      u.Name,
		Email:     "",
		Role:      admin.RoleUser,
		Status:    status,
		Platform:  "local",
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (ac *AdminController) listUsers(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Get users from database
	dbUsers, err := dao.User.WithContext(ctx.Context()).Find()
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}

	all := make([]admin.User, 0, len(dbUsers))
	for _, u := range dbUsers {
		all = append(all, modelUserToAdminUser(u))
	}

	// Pagination
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

	return ctx.JSON(protocol.NewSuccessResponse(admin.ListResponse[admin.User]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}))
}

func (ac *AdminController) createUser(ctx fiber.Ctx) error {
	var req admin.UserCreateRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if req.Name == "" {
		return protocol.ErrBadParam.New("name is required")
	}

	now := time.Now()
	newUser := &model.User{
		Flag:      uuid.New().String(),
		Name:      req.Name,
		State:     model.UserActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := dao.User.WithContext(ctx.Context()).Create(newUser); err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelUserToAdminUser(newUser)))
}

func (ac *AdminController) getUser(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	u, err := dao.User.WithContext(ctx.Context()).Where(dao.User.ID.Eq(id)).First()
	if err != nil {
		return protocol.ErrNotFound.New("user not found")
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelUserToAdminUser(u)))
}

func (ac *AdminController) updateUser(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	var req admin.UserUpdateRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	u, err := dao.User.WithContext(ctx.Context()).Where(dao.User.ID.Eq(id)).First()
	if err != nil {
		return protocol.ErrNotFound.New("user not found")
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	u.UpdatedAt = time.Now()

	if err := dao.User.WithContext(ctx.Context()).Save(u); err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelUserToAdminUser(u)))
}

func (ac *AdminController) deleteUser(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	_, err = dao.User.WithContext(ctx.Context()).Where(dao.User.ID.Eq(id)).Delete()
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

// ---------------------------------------------------------------------------
// Workflow management API
// ---------------------------------------------------------------------------

func modelWorkflowToAdminWorkflow(w *model.Workflow) admin.Workflow {
	status := admin.WorkflowPending
	switch w.State {
	case model.WorkflowEnable:
		status = admin.WorkflowRunning
	case model.WorkflowDisable:
		status = admin.WorkflowPending
	}

	return admin.Workflow{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Describe,
		Status:      status,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

func (ac *AdminController) listWorkflows(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Get workflows from database
	dbWorkflows, err := dao.Workflow.WithContext(ctx.Context()).Find()
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}

	all := make([]admin.Workflow, 0, len(dbWorkflows))
	for _, w := range dbWorkflows {
		all = append(all, modelWorkflowToAdminWorkflow(w))
	}

	// Sort by created_at desc
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	// Pagination
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

	return ctx.JSON(protocol.NewSuccessResponse(admin.ListResponse[admin.Workflow]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}))
}

func (ac *AdminController) createWorkflow(ctx fiber.Ctx) error {
	var req admin.WorkflowCreateRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	if req.Name == "" {
		return protocol.ErrBadParam.New("workflow name is required")
	}

	now := time.Now()
	newWorkflow := &model.Workflow{
		UID:       uuid.New().String(),
		Topic:     "default",
		Flag:      uuid.New().String(),
		Name:      req.Name,
		Describe:  req.Description,
		State:     model.WorkflowEnable,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := dao.Workflow.WithContext(ctx.Context()).Create(newWorkflow); err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelWorkflowToAdminWorkflow(newWorkflow)))
}

func (ac *AdminController) getWorkflow(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	w, err := dao.Workflow.WithContext(ctx.Context()).Where(dao.Workflow.ID.Eq(id)).First()
	if err != nil {
		return protocol.ErrNotFound.New("workflow not found")
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelWorkflowToAdminWorkflow(w)))
}

func (ac *AdminController) deleteWorkflow(ctx fiber.Ctx) error {
	id, err := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if err != nil {
		return protocol.ErrBadParam.Wrap(err)
	}

	_, err = dao.Workflow.WithContext(ctx.Context()).Where(dao.Workflow.ID.Eq(id)).Delete()
	if err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(nil))
}

func (ac *AdminController) runWorkflow(ctx fiber.Ctx) error {
	return protocol.ErrUnsupported.New("workflow run is not supported via admin API")
}

// ---------------------------------------------------------------------------
// Bot management API
// ---------------------------------------------------------------------------

func modelBotToAdminBot(b *model.Bot) admin.BotInfo {
	enabled := b.State == model.BotActive
	return admin.BotInfo{
		Name:        b.Name,
		Enabled:     enabled,
		Description: "",
		Commands:    []string{},
		HasForm:     false,
		HasCron:     false,
		HasWebhook:  false,
	}
}

func (ac *AdminController) listBots(ctx fiber.Ctx) error {
	// Get bots from database
	dbBots, err := dao.Bot.WithContext(ctx.Context()).Find()
	if err != nil {
		return protocol.ErrDatabaseReadError.Wrap(err)
	}

	bots := make([]admin.BotInfo, 0, len(dbBots))
	for _, b := range dbBots {
		bots = append(bots, modelBotToAdminBot(b))
	}

	return ctx.JSON(protocol.NewSuccessResponse(admin.BotListResponse{
		Items: bots,
		Total: int64(len(bots)),
	}))
}

func (ac *AdminController) getBot(ctx fiber.Ctx) error {
	name := ctx.Params("name")

	b, err := dao.Bot.WithContext(ctx.Context()).Where(dao.Bot.Name.Eq(name)).First()
	if err != nil {
		return protocol.ErrNotFound.New("bot not found")
	}

	return ctx.JSON(protocol.NewSuccessResponse(modelBotToAdminBot(b)))
}

func (ac *AdminController) enableBot(ctx fiber.Ctx) error {
	name := ctx.Params("name")

	b, err := dao.Bot.WithContext(ctx.Context()).Where(dao.Bot.Name.Eq(name)).First()
	if err != nil {
		return protocol.ErrNotFound.New("bot not found")
	}

	b.State = model.BotActive
	if err := dao.Bot.WithContext(ctx.Context()).Save(b); err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{"status": "enabled"}))
}

func (ac *AdminController) disableBot(ctx fiber.Ctx) error {
	name := ctx.Params("name")

	b, err := dao.Bot.WithContext(ctx.Context()).Where(dao.Bot.Name.Eq(name)).First()
	if err != nil {
		return protocol.ErrNotFound.New("bot not found")
	}

	b.State = model.BotInactive
	if err := dao.Bot.WithContext(ctx.Context()).Save(b); err != nil {
		return protocol.ErrDatabaseWriteError.Wrap(err)
	}

	return ctx.JSON(protocol.NewSuccessResponse(map[string]string{"status": "disabled"}))
}

// ---------------------------------------------------------------------------
// Log viewer API (mock data)
// ---------------------------------------------------------------------------

var (
	mockLogs     []admin.LogEntry
	mockLogID    atomic.Int64
	logsInitOnce sync.Once
)

func initMockLogs() {
	logsInitOnce.Do(func() {
		now := time.Now()
		mockLogs = []admin.LogEntry{
			{ID: 1, Level: admin.LogLevelInfo, Message: "Server started successfully", Source: "server", Timestamp: now.Add(-1 * time.Hour).Format(time.RFC3339)},
			{ID: 2, Level: admin.LogLevelInfo, Message: "Connected to database", Source: "server", Timestamp: now.Add(-59 * time.Minute).Format(time.RFC3339)},
			{ID: 3, Level: admin.LogLevelWarn, Message: "High memory usage detected: 85%", Source: "server", Timestamp: now.Add(-30 * time.Minute).Format(time.RFC3339)},
			{ID: 4, Level: admin.LogLevelInfo, Message: "Workflow 'Daily Report' completed", Source: "workflow", Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339)},
			{ID: 5, Level: admin.LogLevelError, Message: "Failed to connect to Slack API: timeout", Source: "platform", Timestamp: now.Add(-10 * time.Minute).Format(time.RFC3339)},
			{ID: 6, Level: admin.LogLevelDebug, Message: "Processing message: hello world", Source: "agent", Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339)},
			{ID: 7, Level: admin.LogLevelInfo, Message: "User login: admin@flowbot.io", Source: "server", Timestamp: now.Add(-2 * time.Minute).Format(time.RFC3339)},
			{ID: 8, Level: admin.LogLevelWarn, Message: "Slow query detected: 2.5s", Source: "server", Timestamp: now.Add(-1 * time.Minute).Format(time.RFC3339)},
		}
		mockLogID.Store(9)
	})
}

func (ac *AdminController) listLogs(ctx fiber.Ctx) error {
	initMockLogs()

	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Query("page_size", "50"))
	level := ctx.Query("level")
	source := ctx.Query("source")
	search := ctx.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 50
	}

	ac.mu.RLock()
	all := make([]admin.LogEntry, len(mockLogs))
	copy(all, mockLogs)
	ac.mu.RUnlock()

	if level != "" {
		filtered := make([]admin.LogEntry, 0)
		for _, l := range all {
			if strings.EqualFold(string(l.Level), level) {
				filtered = append(filtered, l)
			}
		}
		all = filtered
	}

	if source != "" {
		filtered := make([]admin.LogEntry, 0)
		for _, l := range all {
			if strings.EqualFold(l.Source, source) {
				filtered = append(filtered, l)
			}
		}
		all = filtered
	}

	if search != "" {
		filtered := make([]admin.LogEntry, 0)
		searchLower := strings.ToLower(search)
		for _, l := range all {
			if strings.Contains(strings.ToLower(l.Message), searchLower) {
				filtered = append(filtered, l)
			}
		}
		all = filtered
	}

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

	return ctx.JSON(protocol.NewSuccessResponse(admin.LogListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}))
}

func (ac *AdminController) getLogSources(ctx fiber.Ctx) error {
	sources := []string{"server", "agent", "workflow", "platform"}
	return ctx.JSON(protocol.NewSuccessResponse(sources))
}
