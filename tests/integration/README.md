# Integration Tests

This directory contains full integration tests using Testcontainers. These tests spin up real MySQL and Redis containers to test the application against actual dependencies.

## Requirements

- Docker must be running
- Go 1.26+
- Sufficient disk space for container images (~1GB)

## Running Tests

### Run all integration tests
```bash
task test:integration
```

Or directly with go:
```bash
go test -v ./tests/integration/...
```

### Run specific test suite
```bash
go test -v -run TestHealthTestSuite ./tests/integration/...
go test -v -run TestDatabaseTestSuite ./tests/integration/...
```

### Run with short mode (skip integration tests)
```bash
go test -short ./...
```

### Skip integration tests
```bash
SKIP_INTEGRATION_TESTS=true go test ./...
```

## Test Suites

### HealthTestSuite
Tests basic health endpoints and container connectivity:
- Liveness endpoint (`/livez`)
- Readiness endpoint (`/readyz`)
- Startup endpoint (`/startupz`)
- Database connection
- Redis connection
- Container states

### DatabaseTestSuite
Tests database CRUD operations for all major models:
- User CRUD
- Bot CRUD
- Platform CRUD
- Channel CRUD
- Message CRUD
- Webhook CRUD
- Counter CRUD
- Data CRUD
- Config CRUD
- Form CRUD
- Page CRUD
- Behavior CRUD
- Instruct CRUD
- Agent CRUD
- Transaction support
- Transaction rollback

## Architecture

```
tests/integration/
├── suite_test.go         # Base IntegrationTestSuite with Testcontainers
├── health_test.go        # Health endpoint tests
└── database_test.go      # Database operation tests
```

### Base Suite

The `IntegrationTestSuite` in `suite_test.go` provides:

1. **Testcontainers Setup**:
   - MySQL 8.0 container with database `flowbot_test`
   - Redis 7-alpine container
   - Automatic port mapping and connection string generation

2. **Database Initialization**:
   - GORM connection to MySQL
   - Migration file execution using golang-migrate
   - All 50+ migrations applied automatically

3. **Redis Initialization**:
   - go-redis client connection
   - Connection verification

4. **Fiber App Setup**:
   - Configured with production-like middleware
   - Error handling compatible with protocol package
   - Health endpoints registered

5. **Cleanup**:
   - Automatic container termination after tests
   - Database and Redis connection cleanup

## Writing New Integration Tests

To create a new integration test suite:

```go
package integration

import (
    "testing"
)

// MyFeatureTestSuite tests your feature.
type MyFeatureTestSuite struct {
    IntegrationTestSuite
}

// TestSomething tests a specific feature.
func (s *MyFeatureTestSuite) TestSomething() {
    // Use s.DB for database operations
    // Use s.Redis for Redis operations
    // Use s.App for HTTP testing
}

// Test entry point
func TestMyFeatureTestSuite(t *testing.T) {
    suite.Run(t, new(MyFeatureTestSuite))
}
```

## Test Duration

Integration tests take significantly longer than unit tests due to container startup:

- First run: ~60-90 seconds (downloads container images)
- Subsequent runs: ~30-45 seconds (container startup + tests)

## Troubleshooting

### Docker not running
```
Error: Cannot connect to the Docker daemon
```
Start Docker Desktop or Docker daemon.

### Port conflicts
Tests use random port mappings to avoid conflicts. If you encounter issues:
```bash
# Check for running containers
docker ps

# Clean up stopped containers
docker container prune
```

### Slow tests
Container startup is inherently slow. You can:
1. Use `-short` flag to skip integration tests during development
2. Run specific test suites with `-run TestSuiteName`
3. Use `SKIP_INTEGRATION_TESTS=true` to skip entirely

### Migration failures
If migrations fail:
1. Check MySQL container logs: `docker logs <container-id>`
2. Verify migration files exist: `ls pkg/migrate/migrations/`
3. Ensure migrations are up to date: `task migration`

## CI/CD Integration

For CI/CD pipelines, ensure Docker is available:

```yaml
# Example GitHub Actions
- name: Run Integration Tests
  run: task test:integration
```

The tests will automatically pull required images and clean up after execution.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SKIP_INTEGRATION_TESTS` | Skip all integration tests | `false` |
| `MYSQL_IMAGE` | MySQL container image | `mysql:8.0` |
| `REDIS_IMAGE` | Redis container image | `redis:7-alpine` |
