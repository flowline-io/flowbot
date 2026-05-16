//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TestHealthTestSuite runs the health test suite.
func TestHealthTestSuite(t *testing.T) {
	suite.Run(t, new(HealthTestSuite))
}

// HealthTestSuite tests health endpoints with Testcontainers.
type HealthTestSuite struct {
	IntegrationTestSuite
}

// TestHealthEndpoints tests all health check endpoints.
func (s *HealthTestSuite) TestHealthEndpoints() {
	testCases := []struct {
		name       string
		endpoint   string
		expectCode int
	}{
		{
			name:       "liveness endpoint returns 200",
			endpoint:   "/livez",
			expectCode: http.StatusOK,
		},
		{
			name:       "readiness endpoint returns 200",
			endpoint:   "/readyz",
			expectCode: http.StatusOK,
		},
		{
			name:       "startup endpoint returns 200",
			endpoint:   "/startupz",
			expectCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			req := s.MakeRequest(http.MethodGet, tc.endpoint, nil)
			resp, err := s.App.Test(req)
			s.Require().NoError(err)
			s.Equal(tc.expectCode, resp.StatusCode)
		})
	}
}

// TestDatabaseConnection verifies database connection is working.
func (s *HealthTestSuite) TestDatabaseConnection() {
	err := s.DB.Ping()
	s.NoError(err, "database should be accessible")
}

// TestRedisConnection verifies Redis connection is working.
func (s *HealthTestSuite) TestRedisConnection() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	// Set a test key
	err := s.Redis.Set(ctx, "test:key", "value", time.Minute).Err()
	s.NoError(err, "should be able to set Redis key")

	// Get the test key
	val, err := s.Redis.Get(ctx, "test:key").Result()
	s.NoError(err, "should be able to get Redis key")
	s.Equal("value", val)

	// Delete the test key
	err = s.Redis.Del(ctx, "test:key").Err()
	s.NoError(err, "should be able to delete Redis key")
}

// TestContainersAreRunning verifies Testcontainers are running.
func (s *HealthTestSuite) TestContainersAreRunning() {
	s.NotNil(s.pgC, "PostgreSQL container should be running")
	s.NotNil(s.redisC, "Redis container should be running")

	// Check PostgreSQL container state
	state, err := s.pgC.State(s.ctx)
	s.NoError(err)
	s.True(state.Running, "PostgreSQL container should be in running state")

	// Check Redis container state
	state, err = s.redisC.State(s.ctx)
	s.NoError(err)
	s.True(state.Running, "Redis container should be in running state")
}

// TestDatabaseMigrations verifies migrations were applied.
func (s *HealthTestSuite) TestDatabaseMigrations() {
	var exists bool
	err := s.DB.QueryRow(
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')",
	).Scan(&exists)
	s.NoError(err, "should be able to query information_schema")
	s.True(exists, "users table should exist")
}

// TestSuiteInheritance verifies that the test suite is properly initialized.
func (s *HealthTestSuite) TestSuiteInheritance() {
	// This test ensures all dependencies are initialized
	assert.NotNil(s.T(), s.DB, "database should be initialized")
	assert.NotNil(s.T(), s.Redis, "Redis should be initialized")
	assert.NotNil(s.T(), s.App, "Fiber app should be initialized")
	assert.NotEmpty(s.T(), s.PGDSN, "PostgreSQL DSN should be set")
	assert.NotEmpty(s.T(), s.RedisAddr, "Redis address should be set")
}
