//go:build integration
// +build integration

// Package integration provides full integration tests using Testcontainers.
// These tests require Docker to be running.
package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	storeMigrate "github.com/flowline-io/flowbot/pkg/migrate"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/redis/go-redis/v9"
	"github.com/samber/oops"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// IntegrationTestSuite is the base test suite for all integration tests.
// It manages Testcontainers for MySQL and Redis and provides a configured Fiber app.
type IntegrationTestSuite struct {
	suite.Suite
	ctx       context.Context
	mysqlC    *tcmysql.MySQLContainer
	redisC    testcontainers.Container
	App       *fiber.App
	DB        *gorm.DB
	Redis     *redis.Client
	MySQLDSN  string
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

	// Start MySQL container
	mysqlImage := os.Getenv("MYSQL_IMAGE")
	if mysqlImage == "" {
		mysqlImage = "mysql:8.0"
	}
	mysqlC, err := tcmysql.Run(s.ctx, mysqlImage,
		tcmysql.WithDatabase("flowbot_test"),
		tcmysql.WithUsername("test"),
		tcmysql.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").WithStartupTimeout(60*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start MySQL container")
	s.mysqlC = mysqlC

	// Get MySQL connection string
	mysqlConnStr, err := mysqlC.ConnectionString(s.ctx)
	s.Require().NoError(err, "failed to get MySQL connection string")
	s.MySQLDSN = mysqlConnStr + "?charset=utf8mb4&parseTime=True&loc=Local"

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
	// Port is returned as "port/protocol" (e.g., "32774/tcp")
	redisPortStr := strings.TrimSuffix(fmt.Sprintf("%s", redisPort), "/tcp")
	s.RedisAddr = fmt.Sprintf("%s:%s", redisHost, redisPortStr)

	s.T().Logf("MySQL DSN: %s", s.MySQLDSN)
	s.T().Logf("Redis address: %s", s.RedisAddr)

	// Connect to MySQL with GORM
	s.DB = s.setupDatabase(s.MySQLDSN)

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
	if s.DB != nil {
		sqlDB, _ := s.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	if s.redisC != nil {
		_ = s.redisC.Terminate(s.ctx)
	}
	if s.mysqlC != nil {
		_ = s.mysqlC.Terminate(s.ctx)
	}
}

// setupDatabase connects to MySQL and runs migrations.
func (s *IntegrationTestSuite) setupDatabase(dsn string) *gorm.DB {
	// Open connection with GORM
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	s.Require().NoError(err, "failed to connect to database")

	// Run migrations
	s.runMigrations(db)

	s.T().Log("Database connection established successfully")
	return db
}

// runMigrations runs database migrations.
func (s *IntegrationTestSuite) runMigrations(db *gorm.DB) {
	sqlDB, err := db.DB()
	s.Require().NoError(err)

	driver, err := migratemysql.WithInstance(sqlDB, &migratemysql.Config{})
	s.Require().NoError(err)

	d, err := iofs.New(storeMigrate.Fs, "migrations")
	s.Require().NoError(err)

	m, err := migrate.NewWithInstance("iofs", d, "mysql", driver)
	s.Require().NoError(err)

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		s.Require().NoError(err, "failed to run migrations")
	}

	s.T().Log("Database migrations completed successfully")
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
func (s *IntegrationTestSuite) RequireDB() *gorm.DB {
	s.Require().NotNil(s.DB, "database connection not available")
	return s.DB
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
