//go:build integration
// +build integration

package specs

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/samber/oops"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// configBundle carries container connection details from process 1 to all processes.
type configBundle struct {
	BaseDSN   string `json:"base_dsn"`
	RedisAddr string `json:"redis_addr"`
}

var (
	suiteCtx  = context.Background()
	pgC       *tcpostgres.PostgresContainer
	redisC    testcontainers.Container
	App       *fiber.App
	DB        *sql.DB
	EntClient *gen.Client
	Redis     *redis.Client
	PGDSN     string
	RedisAddr string
)

var _ = SynchronizedBeforeSuite(
	func() []byte {
		suiteCtx = context.Background()

		if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
			Skip("Skipping integration tests: SKIP_INTEGRATION_TESTS=true")
		}

		flog.Init(flog.Config{Level: "info"})

		pgImage := os.Getenv("POSTGRES_IMAGE")
		if pgImage == "" {
			pgImage = "postgres:16-alpine"
		}
		var err error
		pgC, err = tcpostgres.Run(suiteCtx, pgImage,
			tcpostgres.WithUsername("test"),
			tcpostgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithStartupTimeout(60*time.Second),
			),
		)
		Expect(err).NotTo(HaveOccurred(), "failed to start PostgreSQL container")

		pgConnStr, err := pgC.ConnectionString(suiteCtx)
		Expect(err).NotTo(HaveOccurred())
		baseDSN := ensureSSLMode(pgConnStr)

		redisImage := os.Getenv("REDIS_IMAGE")
		if redisImage == "" {
			redisImage = "redis:7-alpine"
		}
		redisC, err = testcontainers.GenericContainer(suiteCtx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        redisImage,
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   wait.ForListeningPort("6379/tcp"),
			},
			Started: true,
		})
		Expect(err).NotTo(HaveOccurred(), "failed to start Redis container")

		redisHost, err := redisC.Host(suiteCtx)
		Expect(err).NotTo(HaveOccurred())
		redisPort, err := redisC.MappedPort(suiteCtx, "6379")
		Expect(err).NotTo(HaveOccurred())
		redisAddr := fmt.Sprintf("%s:%s", redisHost,
			strings.TrimSuffix(fmt.Sprintf("%s", redisPort), "/tcp"))

		GinkgoWriter.Printf("PostgreSQL base DSN: %s\n", baseDSN)
		GinkgoWriter.Printf("Redis address: %s\n", redisAddr)

		data, err := sonic.Marshal(configBundle{BaseDSN: baseDSN, RedisAddr: redisAddr})
		Expect(err).NotTo(HaveOccurred())
		return data
	},
	func(data []byte) {
		var cfg configBundle
		Expect(sonic.Unmarshal(data, &cfg)).To(Succeed())

		procID := GinkgoParallelProcess()
		dbName := fmt.Sprintf("flowbot_test_%d", procID)
		PGDSN = createPerProcessDatabase(cfg.BaseDSN, dbName)
		DB, EntClient = setupEntClient(PGDSN)
		runMigrations()

		Redis = setupRedis(cfg.RedisAddr, procID)
		RedisAddr = cfg.RedisAddr

		App = setupTestApp()

		// Register pipeline CRUD BDD routes before any test-specific
		// BeforeEach can mount the web module's auth-protected versions.
		mountPipelineRoutes(App)

		GinkgoWriter.Printf("Process %d: database=%s\n", procID, dbName)
	},
)

var _ = SynchronizedAfterSuite(
	func() {
		if EntClient != nil {
			_ = EntClient.Close()
		}
		if DB != nil {
			_ = DB.Close()
		}
		if Redis != nil {
			_ = Redis.Close()
		}
	},
	func() {
		if redisC != nil {
			_ = redisC.Terminate(suiteCtx)
		}
		if pgC != nil {
			_ = pgC.Terminate(suiteCtx)
		}
	},
)

// ensureSSLMode parses the DSN URL and sets sslmode=disable.
func ensureSSLMode(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

// createPerProcessDatabase connects to the base DSN, drops any pre-existing database,
// creates a fresh per-process database, and returns a new DSN pointing to it.
func createPerProcessDatabase(baseDSN, dbName string) string {
	adminDB, err := sql.Open("pgx", baseDSN)
	Expect(err).NotTo(HaveOccurred())
	defer adminDB.Close()

	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	Expect(err).NotTo(HaveOccurred())

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	Expect(err).NotTo(HaveOccurred())

	u, err := url.Parse(baseDSN)
	Expect(err).NotTo(HaveOccurred())
	u.Path = "/" + dbName
	return u.String()
}

// setupEntClient opens a database connection and creates an Ent client.
func setupEntClient(dsn string) (*sql.DB, *gen.Client) {
	rawDB, err := sql.Open("pgx", dsn)
	Expect(err).NotTo(HaveOccurred())

	drv := entsql.OpenDB(dialect.Postgres, rawDB)
	client := gen.NewClient(gen.Driver(drv))
	return rawDB, client
}

// runMigrations applies the Ent schema to the per-process database.
func runMigrations() {
	mctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	Expect(EntClient.Schema.Create(mctx)).To(Succeed())
}

// setupRedis connects to Redis using the given address and per-process DB number.
func setupRedis(addr string, db int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Expect(client.Ping(ctx).Err()).To(Succeed())
	return client
}

// setupTestApp creates a configured Fiber app for testing.
func setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		BodyLimit:    20 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if oopsErr, ok := oops.AsOops(err); ok {
				if oopsErr.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return c.Status(fiber.StatusUnauthorized).
						JSON(protocol.NewFailedResponse(oopsErr))
				}
				return c.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(oopsErr))
			}
			if err != nil {
				return c.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.Wrap(err)))
			}
			return nil
		},
	})

	app.Use(recover.New())

	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())

	return app
}
