//go:build integration
// +build integration

// Package integration provides full integration tests using Testcontainers.
// These tests require Docker to be running.
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"github.com/samber/oops"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

// IntegrationTestSuite is the base test suite for all integration tests.
// It manages Testcontainers for PostgreSQL and Redis and provides a configured Fiber app.
type IntegrationTestSuite struct {
	suite.Suite
	ctx       context.Context
	pgC       *tcpostgres.PostgresContainer
	redisC    testcontainers.Container
	App       *fiber.App
	DB        *sql.DB
	EntClient *gen.Client
	Redis     *redis.Client
	PGDSN     string
	RedisAddr string
}

// SetupSuite initializes the test environment with Testcontainers.
func (s *IntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()

	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		s.T().Skip("Skipping integration tests: SKIP_INTEGRATION_TESTS=true")
	}

	// Initialize logging
	flog.Init(flog.Config{Level: "info"})

	// Start PostgreSQL container
	pgImage := os.Getenv("POSTGRES_IMAGE")
	if pgImage == "" {
		pgImage = "postgres:16-alpine"
	}
	pgC, err := tcpostgres.Run(s.ctx, pgImage,
		tcpostgres.WithDatabase("flowbot_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start PostgreSQL container")
	s.pgC = pgC

	// Get PostgreSQL connection string
	pgConnStr, err := pgC.ConnectionString(s.ctx)
	s.Require().NoError(err, "failed to get PostgreSQL connection string")
	s.PGDSN = strings.TrimRight(pgConnStr, "?") + "?sslmode=disable"

	// Start Redis container
	redisImage := os.Getenv("REDIS_IMAGE")
	if redisImage == "" {
		redisImage = "redis:7-alpine"
	}
	redisC, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        redisImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	s.Require().NoError(err, "failed to start Redis container")
	s.redisC = redisC

	// Get Redis connection details
	redisHost, err := redisC.Host(s.ctx)
	s.Require().NoError(err)
	redisPort, err := redisC.MappedPort(s.ctx, "6379")
	s.Require().NoError(err)

	// Extract port number from "port/protocol" format
	redisPortStr := strings.TrimSuffix(fmt.Sprintf("%s", redisPort), "/tcp")
	s.RedisAddr = fmt.Sprintf("%s:%s", redisHost, redisPortStr)

	s.T().Logf("PostgreSQL DSN: %s", s.PGDSN)
	s.T().Logf("Redis address: %s", s.RedisAddr)

	// Connect to PostgreSQL with Ent
	s.DB, s.EntClient = s.setupDatabase(s.PGDSN)

	// Connect to Redis
	s.Redis = s.setupRedis(s.RedisAddr)

	// Create Fiber app
	s.App = s.setupTestApp()
}

// TearDownSuite cleans up testcontainers.
func (s *IntegrationTestSuite) TearDownSuite() {
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
	if s.EntClient != nil {
		_ = s.EntClient.Close()
	}
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.redisC != nil {
		_ = s.redisC.Terminate(s.ctx)
	}
	if s.pgC != nil {
		_ = s.pgC.Terminate(s.ctx)
	}
}

// setupDatabase connects to PostgreSQL, creates the Ent client, and runs schema migrations.
func (s *IntegrationTestSuite) setupDatabase(dsn string) (*sql.DB, *gen.Client) {
	rawDB, err := sql.Open("pgx", dsn)
	s.Require().NoError(err, "failed to connect to database")

	drv := entsql.OpenDB(dialect.Postgres, rawDB)
	client := gen.NewClient(gen.Driver(drv))

	mctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	err = client.Schema.Create(mctx)
	s.Require().NoError(err, "failed to run schema migrations")

	s.T().Log("Database connection established successfully")
	return rawDB, client
}

// setupRedis connects to Redis.
func (s *IntegrationTestSuite) setupRedis(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	s.Require().NoError(err, "failed to connect to Redis")

	s.T().Log("Redis connection established successfully")
	return client
}

// setupTestApp creates a configured Fiber app for testing.
func (s *IntegrationTestSuite) setupTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			if oopsErr, ok := oops.AsOops(err); ok {
				if oopsErr.Code() == oops.OopsError(protocol.ErrNotAuthorized).Code() {
					return ctx.Status(fiber.StatusUnauthorized).
						JSON(protocol.NewFailedResponse(oopsErr))
				}
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(oopsErr))
			}
			if err != nil {
				return ctx.Status(fiber.StatusBadRequest).
					JSON(protocol.NewFailedResponse(protocol.ErrBadRequest.Wrap(err)))
			}
			return nil
		},
	})

	// Recovery middleware
	app.Use(recover.New())

	// Health endpoints
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())

	return app
}

// RequireDB returns the database connection and fails the test if not available.
func (s *IntegrationTestSuite) RequireDB() *sql.DB {
	s.Require().NotNil(s.DB, "database connection not available")
	return s.DB
}

// RequireEntClient returns the Ent client and fails the test if not available.
func (s *IntegrationTestSuite) RequireEntClient() *gen.Client {
	s.Require().NotNil(s.EntClient, "Ent client not available")
	return s.EntClient
}

// RequireRedis returns the Redis client and fails the test if not available.
func (s *IntegrationTestSuite) RequireRedis() *redis.Client {
	s.Require().NotNil(s.Redis, "Redis connection not available")
	return s.Redis
}

// RequireApp returns the Fiber app and fails the test if not available.
func (s *IntegrationTestSuite) RequireApp() *fiber.App {
	s.Require().NotNil(s.App, "Fiber app not available")
	return s.App
}

// MakeRequest creates an HTTP request for testing.
func (s *IntegrationTestSuite) MakeRequest(method, path string, body []byte) *http.Request {
	var bodyReader *strings.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}

	// Use a full URL format for Fiber testing (http://localhost is required for Host header)
	url := "http://localhost" + path
	req, err := http.NewRequest(method, url, bodyReader)
	s.Require().NoError(err)
	return req
}

// JSONRequest creates a JSON HTTP request for testing.
func (s *IntegrationTestSuite) JSONRequest(method, path string, body []byte) *http.Request {
	req := s.MakeRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// AssertSuccessResponse asserts that the response indicates success.
func (s *IntegrationTestSuite) AssertSuccessResponse(resp *http.Response) {
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	body := s.ReadBody(resp)
	s.Contains(string(body), `"status":"success"`)
}

// ReadBody reads and returns the response body.
func (s *IntegrationTestSuite) ReadBody(resp *http.Response) []byte {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	return body
}
