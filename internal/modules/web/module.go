// Package web provides a web UI module with server-rendered HTML pages.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	webassets "github.com/flowline-io/flowbot"
	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/webauth"
)

const Name = "web"

var handler moduleHandler
var config configType

// Register registers the web module handler.
func Register() {
	module.Register(Name, &handler)
	lifemod.OnService(SetLifeService)
}

type moduleHandler struct {
	initialized bool
	authConfig  AuthConfig
	module.Base
}

type configType struct {
	Enabled bool       `json:"enabled"`
	Auth    AuthConfig `json:"auth"`
}

// Init initializes the web module with the given JSON configuration.
func (moduleHandler) Init(jsonconf json.RawMessage) error {
	if handler.initialized {
		return errors.New("already initialized")
	}
	if err := sonic.Unmarshal(jsonconf, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if !config.Enabled {
		flog.Info("module %s disabled", Name)
		return nil
	}
	if err := validateAuthConfig(config.Auth); err != nil {
		return err
	}
	config.Auth.BruteForce.applyDefaults()

	keyDir := config.Auth.EncryptionDir
	if keyDir == "" {
		keyDir = "."
	}
	enc, fromFile, created, err := webauth.LoadEncryptor(config.Auth.EncryptionKey, keyDir)
	if err != nil {
		return fmt.Errorf("web auth encryption key: %w", err)
	}
	if fromFile {
		if created {
			flog.Warn("web auth: generated encryption key file under %s (prefer modules.web.auth.encryption_key via env)", keyDir)
		} else {
			flog.Warn("web auth: using encryption key file under %s (prefer modules.web.auth.encryption_key via env)", keyDir)
		}
	}
	setWebEncryptor(enc)

	handler.initialized = true
	handler.authConfig = config.Auth
	wireLoginRateLimiter()
	return nil
}

// IsReady checks if the web module is initialized.
func (moduleHandler) IsReady() bool {
	return handler.initialized
}

// Bootstrap performs post-initialization setup including YAML→DB credential migration.
func (moduleHandler) Bootstrap() error {
	if !handler.initialized {
		return nil
	}
	if err := migrateYAMLCredentials(); err != nil {
		return err
	}
	if n, err := store.WebAccountStoreFromDB().RevokeLegacyWebSessions(context.Background()); err != nil {
		flog.Error(fmt.Errorf("web auth: revoke legacy sessions: %w", err))
	} else if n > 0 {
		flog.Info("web auth: revoked %d legacy or pending web session(s); re-login required", n)
	}
	warnResidualYAMLAuth()
	if store.Database != nil {
		es := store.EventStoreFromDB()
		sources, err := es.ListDistinctEventSources(context.Background(), 30*24*time.Hour)
		if err == nil {
			distinctTypes, err2 := es.ListDistinctEventTypes(context.Background(), 30*24*time.Hour)
			if err2 == nil {
				types.EventFilterCache.Hydrate(sources, distinctTypes)
			}
		}
	}
	return nil
}

func migrateYAMLCredentials() error {
	ws := store.WebAccountStoreFromDB()
	n, err := ws.Count(context.Background())
	if err != nil {
		flog.Debug("web auth: skip yaml migration: %v", err)
		return nil
	}
	if n > 0 {
		return nil
	}
	cfg := config.Auth
	if strings.TrimSpace(cfg.Username) == "" || (cfg.Password == "" && cfg.PasswordHash == "") {
		flog.Info("web auth: no accounts in database; use /service/web/setup to create the first admin")
		return nil
	}
	hash, err := yamlMigrationHash(cfg)
	if err != nil {
		return fmt.Errorf("web auth migrate: %w", err)
	}
	if _, err := ws.CreateFirstAccount(context.Background(), store.CreateAccountInput{
		Username:     cfg.Username,
		PasswordHash: hash,
	}); err != nil {
		return fmt.Errorf("web auth migrate: %w", err)
	}
	flog.Info("web auth: migrated YAML credentials for user %q into database; remove password from flowbot.yaml", cfg.Username)
	return nil
}

func warnResidualYAMLAuth() {
	ws := store.WebAccountStoreFromDB()
	n, err := ws.Count(context.Background())
	if err != nil || n == 0 {
		return
	}
	cfg := config.Auth
	if cfg.Password != "" {
		flog.Warn("web auth: modules.web.auth.password is still set in config but accounts live in the database; remove the plaintext password")
	}
	if cfg.PasswordHash != "" {
		flog.Warn("web auth: modules.web.auth.password_hash is still set in config but accounts live in the database; remove it after migration")
	}
}

// Webservice mounts web module routes on the fiber app.
func (moduleHandler) Webservice(app *fiber.App) {
	app.Get("/static/*", static.New("", static.Config{FS: webassets.SubFS}))
	app.Use("/c", localeMiddleware())
	app.Get("/c/:slug", clipPage)
	app.Use("/service/web", localeMiddleware())
	app.Use("/service/web", newCSRFMiddleware())
	for _, rules := range allWebserviceRules {
		module.Webservice(app, Name, rules)
	}
}

// Rules returns the web module rule definitions as one entry per route group in allWebserviceRules.
func (moduleHandler) Rules() []any {
	out := make([]any, len(allWebserviceRules))
	for i, rules := range allWebserviceRules {
		out[i] = rules
	}
	return out
}

// InitForE2E initializes the web module handler for e2e testing.
// It calls the package-level handler.Init with the provided JSON config,
// bypassing the uber/fx dependency injection used in production.
func InitForE2E(configData json.RawMessage) error {
	return handler.Init(configData)
}

// MountForE2E mounts web module routes onto the given Fiber app.
func MountForE2E(app *fiber.App) {
	ensureChatAgentService()
	handler.Webservice(app)
}
