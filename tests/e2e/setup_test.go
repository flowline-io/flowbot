//go:build e2e

// Package e2e provides browser-based end-to-end tests for the web module.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/samber/oops"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/parameter"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

var (
	browser        *rod.Browser
	baseURL        string
	app            *fiber.App
	pgContainer    *tcpostgres.PostgresContainer
	redisContainer testcontainers.Container
	testToken      string
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	flog.Init(flog.Config{Level: "info"})

	var err error

	pgContainer, err = tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres: %v\n", err)
		return 1
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres: %v\n", err)
		}
	}()

	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres connection string: %v\n", err)
		return 1
	}

	redisContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start redis: %v\n", err)
		return 1
	}
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate redis: %v\n", err)
		}
	}()

	postgres.Init()
	store.Init()

	storeCfg := config.StoreType{
		Adapters: map[string]any{
			"postgres": map[string]any{
				"dsn": pgDSN,
			},
		},
	}
	if err := store.Store.Open(storeCfg); err != nil {
		fmt.Fprintf(os.Stderr, "store open: %v\n", err)
		return 1
	}
	defer func() {
		if err := store.Store.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "store close: %v\n", err)
		}
	}()

	if err := store.Migrate(); err != nil {
		fmt.Fprintf(os.Stderr, "store migrate: %v\n", err)
		return 1
	}

	token := "fb_e2e-test-token-" + time.Now().Format("20060102150405")
	params := types.KV{
		"uid":    "user-admin",
		"topic":  "web",
		"scopes": []string{"admin:*"},
	}
	err = store.Database.ParameterSet(context.Background(), token, params, time.Now().Add(24*time.Hour))
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed token: %v\n", err)
		return 1
	}
	testToken = token

	app = fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		BodyLimit:    20 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if oopsErr, ok := oops.AsOops(err); ok {
				if oopsErr.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return c.Status(fiber.StatusUnauthorized).JSON(protocol.NewFailedResponse(oopsErr))
				}
				return c.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(oopsErr))
			}
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.Wrap(err)))
			}
			return nil
		},
	})
	app.Use(recover.New())
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())

	webConfig := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"e2e-test-pass"}}`)
	if err := web.InitForE2E(webConfig); err != nil {
		fmt.Fprintf(os.Stderr, "web init: %v\n", err)
		return 1
	}
	web.MountForE2E(app)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		return 1
	}
	baseURL = "http://" + l.Addr().String()

	go func() {
		if err := app.Listener(l); err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		}
	}()
	defer app.Shutdown()

	launcherURL := launcher.New().NoSandbox(true).MustLaunch()
	browser = rod.New().ControlURL(launcherURL).MustConnect()
	defer browser.MustClose()

	return m.Run()
}

// NewPage creates a new incognito browser page with cleanup and failure screenshot.
func NewPage(t testing.TB) *rod.Page {
	t.Helper()
	page := browser.MustIncognito().MustPage()
	t.Cleanup(func() {
		if t.Failed() {
			dir := "test-reports"
			if err := os.MkdirAll(dir, 0755); err == nil {
				page.MustScreenshot(dir + "/" + t.Name() + ".png")
			}
		}
		page.MustClose()
	})
	return page
}

// URL returns the full URL for the given path.
func URL(path string) string {
	return baseURL + path
}

// loginViaCookie injects an accessToken cookie to authenticate without UI login.
func loginViaCookie(t *testing.T) *rod.Page {
	t.Helper()
	page := NewPage(t)
	page.MustSetCookies(&proto.NetworkCookieParam{
		Name:  "accessToken",
		Value: testToken,
		URL:   baseURL,
	})
	return page
}

// seedConfig creates a config entry directly via the store adapter.
func seedConfig(t *testing.T, uid, topic, key string, value interface{}) {
	t.Helper()
	err := store.Database.ConfigSet(context.Background(), types.Uid(uid), topic, key, types.KV{"value": value})
	if err != nil {
		t.Fatalf("seedConfig: %v", err)
	}
}

// ResetDB truncates config, parameter, and pipeline tables (except the test token).
// Call at the top of each CRUD test case to prevent state bleeding.
func ResetDB(t *testing.T) {
	t.Helper()
	client := store.Database.GetDB().(*gen.Client)
	ctx := context.Background()

	_, err := client.ConfigData.Delete().Exec(ctx)
	if err != nil {
		t.Fatalf("reset db configdata: %v", err)
	}
	_, err = client.Parameter.Delete().Where(parameter.FlagNEQ(testToken)).Exec(ctx)
	if err != nil {
		t.Fatalf("reset db parameter: %v", err)
	}

	_, err = client.PipelineStepRun.Delete().Exec(ctx)
	if err != nil {
		t.Fatalf("reset db pipelinesteprun: %v", err)
	}
	_, err = client.EventConsumption.Delete().Exec(ctx)
	if err != nil {
		t.Fatalf("reset db eventconsumption: %v", err)
	}
	_, err = client.PipelineRun.Delete().Exec(ctx)
	if err != nil {
		t.Fatalf("reset db pipelinerun: %v", err)
	}
	_, err = client.PipelineDefinition.Delete().Exec(ctx)
	if err != nil {
		t.Fatalf("reset db pipelinedefinition: %v", err)
	}
}

// seedPipeline creates a pipeline definition directly via the ent client.
func seedPipeline(t *testing.T, name string) {
	t.Helper()
	client := store.Database.GetDB().(*gen.Client)
	ctx := context.Background()
	now := time.Now()
	_, err := client.PipelineDefinition.Create().
		SetName(name).
		SetYamlDraft("").
		SetNillableYamlPublished(nil).
		SetVersion(1).
		SetStatus("draft").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	require.NoError(t, err)
}
