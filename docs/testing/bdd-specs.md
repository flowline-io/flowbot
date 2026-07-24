# BDD Acceptance Testing with Ginkgo

Flowbot uses Ginkgo v2 + Gomega for Behavior-Driven Development (BDD) at the integration and acceptance level. Unit tests retain testify with table-driven patterns. This document describes the infrastructure, conventions, and workflow.

## Test Pyramid

```
                         ┌───────────────────────────┐
                         │  BDD Acceptance Tests     │
                         │  Ginkgo + Gomega          │
                         │  tests/specs/             │
                         │  Requires Docker          │
                         ├───────────────────────────┤
                         │  Unit Tests               │
                         │  testify table-driven     │
                         │  pkg/** / *_test.go       │
                         │  No external deps         │
                         └───────────────────────────┘
```

| Layer          | Framework            | Location                             | Notes                                     |
| -------------- | -------------------- | ------------------------------------ | ----------------------------------------- |
| BDD Acceptance | Ginkgo + Gomega      | `tests/specs/`                       | All integration/acceptance tests must use |
| Unit           | testify table-driven | `pkg/**/`, `internal/**/`, `cmd/**/` | Never migrate -- retained permanently     |

## When BDD is required

* **Required:** new modules; changes that alter cross-boundary behavior (HTTP APIs, events, auth, pipelines) visible outside a single package.
* **Not required:** pure library refactors covered by unit tests; docs / AGENTS / comment-only edits.
* **Without Docker:** `go tool task test:specs` needs Docker/testcontainers. Run unit tests (`go tool task test`) and explicitly state that BDD specs were skipped — do not claim specs passed.
* Repo-wide policy summary: root [AGENTS.md](../../AGENTS.md) Testing policy.

## Directory Structure

```
tests/
├── specs/                              # Ginkgo BDD tests
│   ├── specs_suite_test.go             # Suite entry point (TestSpecs + RunSpecs)
│   ├── lifecycle.go                    # SynchronizedBeforeSuite / AfterSuite + per-process DB isolation
│   ├── fixtures.go                     # HTTP request helpers (MakeRequest, JSONRequest, ReadBody)
│   ├── ability_spec_test.go            # Ability layer
│   ├── auth_spec_test.go               # Authentication contexts, tokens, scopes
│   ├── bookmark_spec_test.go           # Bookmark module
│   ├── database_spec_test.go           # Core database model CRUD
│   ├── database_ext_spec_test.go       # Extended database model CRUD
│   ├── event_spec_test.go              # DataEvent publish, consume, idempotency
│   ├── example_spec_test.go            # Example module
│   ├── gitea_spec_test.go              # Gitea module
│   ├── github_spec_test.go             # GitHub module
│   ├── health_spec_test.go             # Health checks + smoke tests
│   ├── homelab_spec_test.go            # Homelab scanner
│   ├── hub_spec_test.go                # Hub management
│   ├── kanban_spec_test.go             # Kanban module
│   ├── llm_spec_test.go                # LLM integration
│   ├── agent_spec_test.go              # Agent engine (pkg/agent)
│   ├── notify_spec_test.go             # Notify module
│   ├── pipeline_spec_test.go           # Pipeline engine
│   ├── provider_event_source_spec_test.go # Provider event source
│   ├── reader_spec_test.go             # Reader module
│   ├── server_spec_test.go             # Server module
│   └── workflow_spec_test.go           # Workflow module
```

All files under `tests/specs/` use `//go:build integration` to prevent compilation during standard unit test runs.

## Infrastructure

### Parallel Database Isolation

Ginkgo's `--procs=N` flag runs N independent test processes. To prevent data conflicts, each process operates on an isolated database namespace using `GinkgoParallelProcess()`.

```
                     ┌──────────────────────┐
                     │  Process 1           │
                     │  SBS process1        │  Start PostgreSQL + Redis containers
                     │                      │  Serialize DSN -> all processes
                     └──────────┬───────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          ▼                    ▼                    ▼
   ┌───────────────┐   ┌───────────────┐   ┌───────────────┐
   │  Process 1    │   │  Process 2    │   │  Process 3    │
   │  DB:          │   │  DB:          │   │  DB:          │
   │  flowbot      │   │  flowbot      │   │  flowbot      │
   │  _test_1      │   │  _test_2      │   │  _test_3      │
   │  Redis DB: 1  │   │  Redis DB: 2  │   │  Redis DB: 3  │
   │  Run specs    │   │  Run specs    │   │  Run specs    │
   └───────────────┘   └───────────────┘   └───────────────┘
          │                     │                     │
          └─────────────────────┼─────────────────────┘
                                ▼
                     ┌──────────────────────┐
                     │  Process 1           │
                     │  SAS process1        │  Wait all done
                     │                      │  Terminate containers
                     └──────────────────────┘
```

**Key mechanism**:

1. `SynchronizedBeforeSuite` process 1 starts one PostgreSQL container and one Redis container.
2. Container connection details are serialized and passed to all processes.
3. Each process calls `GinkgoParallelProcess()` to get a unique ID (1, 2, 3, ...).
4. Each process:
   - Connects to the PostgreSQL container and runs `CREATE DATABASE flowbot_test_{ID}`.
   - Creates an Ent client on that database and runs schema migrations.
   - Connects to Redis using `DB: GinkgoParallelProcess()` for key-space isolation.
   - Creates a Fiber app instance for HTTP testing.
5. All processes run their assigned specs in parallel -- zero data conflicts.
6. `SynchronizedAfterSuite` process 1 terminates containers after all processes complete.

### Suite-Level Variables

The following variables are initialized by `lifecycle.go` and available to all spec files in the `specs` package:

| Variable    | Type                            | Purpose                               |
| ----------- | ------------------------------- | ------------------------------------- |
| `suiteCtx`  | `context.Context`               | Context scoped to container lifecycle |
| `pgC`       | `*tcpostgres.PostgresContainer` | PostgreSQL testcontainer              |
| `redisC`    | `testcontainers.Container`      | Redis testcontainer                   |
| `App`       | `*fiber.App`                    | Configured Fiber HTTP app for testing |
| `DB`        | `*sql.DB`                       | Raw database connection               |
| `EntClient` | `*gen.Client`                   | Ent ORM client                        |
| `Redis`     | `*redis.Client`                 | Redis client                          |
| `PGDSN`     | `string`                        | Per-process database DSN              |
| `RedisAddr` | `string`                        | Redis address                         |

### SynchronizedBeforeSuite Lifecycle

```go
// Process 1: start containers, serialize config
var _ = SynchronizedBeforeSuite(
    func() []byte {
        flog.Init(flog.Config{Level: "info"})
        // Start PostgreSQL container
        pgC, _ = tcpostgres.Run(ctx, pgImage,
            tcpostgres.WithUsername("test"),
            tcpostgres.WithPassword("test"),
            testcontainers.WithWaitStrategy(...),
        )
        // Start Redis container
        // ...
        return sonic.Marshal(configBundle{BaseDSN: baseDSN, RedisAddr: redisAddr})
    },

    // All processes: parse config, create per-process database
    func(data []byte) {
        procID := GinkgoParallelProcess()
        dbName := fmt.Sprintf("flowbot_test_%d", procID)
        PGDSN = createPerProcessDatabase(cfg.BaseDSN, dbName)
        DB, EntClient = setupEntClient(PGDSN)
        runMigrations()
        Redis = setupRedis(cfg.RedisAddr, procID)
        App = setupTestApp()
    },
)
```

### createPerProcessDatabase

PostgreSQL requires connecting to an existing database to issue `CREATE DATABASE`. The base DSN from testcontainers points to the default database (typically `test`). The function connects via that DSN, executes `CREATE DATABASE`, and returns a new DSN targeting the per-process database.

```go
func createPerProcessDatabase(baseDSN, dbName string) string {
    adminDB, _ := sql.Open("pgx", baseDSN)
    defer adminDB.Close()

    adminDB.Exec("DROP DATABASE IF EXISTS " + dbName)
    adminDB.Exec("CREATE DATABASE " + dbName)

    u, _ := url.Parse(baseDSN)
    u.Path = "/" + dbName
    return u.String()
}
```

## Writing Specs

### Spec Structure

Ginkgo specs use three node types to build a descriptive hierarchy:

| Node Type | Examples                                                    | Purpose                         |
| --------- | ----------------------------------------------------------- | ------------------------------- |
| Container | `Describe`, `Context`, `When`                               | Organize specs hierarchically   |
| Setup     | `BeforeEach`, `AfterEach`, `JustBeforeEach`, `DeferCleanup` | Set up and tear down spec state |
| Subject   | `It`, `Specify`                                             | Make assertions about behavior  |

### Basic Pattern

```go
var _ = Describe("Bookmark Module", Label("module", "bookmark"), func() {
    var bookmarkID string

    BeforeEach(func() {
        // Shared setup: cleanup, prepare test data
    })

    Describe("creating a bookmark", func() {
        Context("with a valid URL", func() {
            It("stores the bookmark and returns success", func() {
                req := JSONRequest("POST", "/service/bookmark/create",
                    []byte(`{"url":"https://example.com"}`))
                resp, _ := App.Test(req)

                Expect(resp.StatusCode).To(Equal(200))
                Expect(ReadBody(resp)).To(ContainSubstring("success"))
            })
        })

        Context("with an empty URL", func() {
            It("returns a validation error", func() {
                req := JSONRequest("POST", "/service/bookmark/create",
                    []byte(`{"url":""}`))
                resp, _ := App.Test(req)

                Expect(resp.StatusCode).To(Equal(400))
            })
        })
    })
})
```

### Labels

Labels enable selective test execution. Use them to categorize specs:

| Label           | Convention                  | Usage                                     |
| --------------- | --------------------------- | ----------------------------------------- |
| `smoke`         | Fast, high-value tests      | `ginkgo --label-filter="smoke"`           |
| `module:<name>` | Tests for a specific module | `ginkgo --label-filter="module/bookmark"` |
| `integration`   | Requires external services  | `ginkgo --label-filter="integration"`     |

```go
var _ = Describe("Health", Label("health", "smoke"), func() { ... })
```

### DescribeTable (Table-Driven BDD)

`DescribeTable` maps the table-driven pattern to Ginkgo's BDD style. Entries are distributed across parallel processes when `--procs` is active.

```go
DescribeTable("returns 200 for health endpoints",
    func(endpoint string) {
        req := MakeRequest(http.MethodGet, endpoint, nil)
        resp, err := App.Test(req)
        Expect(err).NotTo(HaveOccurred())
        Expect(resp.StatusCode).To(Equal(http.StatusOK))
    },
    Entry("liveness endpoint", "/livez"),
    Entry("readiness endpoint", "/readyz"),
    Entry("startup endpoint", "/startupz"),
)
```

### Assertions

In Ginkgo, `RegisterFailHandler(Fail)` connects Gomega to Ginkgo. When `Expect` fails, it **immediately terminates** the current `It` block (equivalent to `testify/require`, not `testify/assert`). No guard clauses are needed after `Expect(err).NotTo(HaveOccurred())`.

| testify                      | Gomega                                |
| ---------------------------- | ------------------------------------- |
| `require.NoError(t, err)`    | `Expect(err).NotTo(HaveOccurred())`   |
| `require.Equal(t, a, b)`     | `Expect(b).To(Equal(a))`              |
| `require.True(t, cond)`      | `Expect(cond).To(BeTrue())`           |
| `require.NotNil(t, v)`       | `Expect(v).NotTo(BeNil())`            |
| `assert.Equal(t, a, b)`      | `Expect(b).To(Equal(a))`              |
| `assert.Contains(t, s, sub)` | `Expect(s).To(ContainSubstring(sub))` |
| `s.Require().NoError(err)`   | `Expect(err).NotTo(HaveOccurred())`   |
| `s.Equal(a, b)`              | `Expect(b).To(Equal(a))`              |

### HTTP Testing

All HTTP tests use `App.Test(req)` on the shared `*fiber.App` instance -- no local server is started. The `fixtures.go` helpers simplify request construction:

| Function      | Signature                                          |
| ------------- | -------------------------------------------------- |
| `MakeRequest` | `(method, path string, body []byte) *http.Request` |
| `JSONRequest` | `(method, path string, body []byte) *http.Request` |
| `ReadBody`    | `(resp *http.Response) []byte`                     |

When a module or capability might not be registered in the test setup, use `Or(...)` to accept multiple status codes:

```go
Expect(resp.StatusCode).To(Or(
    Equal(http.StatusOK),
    Equal(http.StatusBadRequest),
    Equal(http.StatusUnauthorized),
))
```

Or call `Skip(...)` when a prerequisite is missing:

```go
if err != nil {
    Skip("bookmark capability not registered: " + err.Error())
}
```

### Database Testing

Database specs use the shared `EntClient` directly. CRUD pattern: **Create -> Assert -> Cleanup**. Each test creates unique records using `types.Id()` for random suffixes.

```go
It("creates a new user with valid data", func() {
    u, err := EntClient.User.Create().
        SetFlag("test-flag-" + types.Id()).
        SetName("Test User").
        Save(ctx)
    Expect(err).NotTo(HaveOccurred())
    Expect(u.ID).NotTo(BeZero())

    EntClient.User.DeleteOne(u).Exec(ctx)
})
```

## Running Specs

### Task Commands

```bash
# Local parallel run (requires Docker)
go tool task test:specs

# CI run with flake retry and JUnit output
go tool task test:specs:ci

# Serial run for debugging
go tool task test:specs:serial

# Smoke tests only
go tool ginkgo --label-filter="smoke" --tags integration ./tests/specs/...

# Specific module
go tool ginkgo --label-filter="module/bookmark" --tags integration ./tests/specs/...
```

### Taskfile Definitions

```yaml
test:specs:
  desc: Run Ginkgo BDD acceptance tests (parallel, requires Docker)
  cmds:
    - go tool ginkgo --procs=4 -v --randomize-all --fail-fast --tags integration ./tests/specs/...

test:specs:ci:
  desc: Run BDD tests for CI with retry and JUnit output
  cmds:
    - go tool ginkgo --procs=4 --flake-attempts=2 --junit-report=specs-report.xml --tags integration ./tests/specs/...

test:specs:serial:
  desc: Run BDD tests serially for debugging
  cmds:
    - go tool ginkgo -v --trace --tags integration ./tests/specs/...
```

### go test Compatibility

Ginkgo suites are standard Go tests (entry point is `func TestSpecs(t *testing.T)`). Standard tooling works:

```bash
# Standard go test (no parallel within package)
go test -v -tags integration ./tests/specs/...

# gotestsum
go tool gotestsum -- -tags integration ./tests/specs/...

# Coverage
go test -tags integration -coverprofile=coverage.out ./tests/specs/...
go tool cover -html=coverage.out
```

`ginkgo` CLI is recommended over `go test` for its spec randomization, parallel process distribution, flake retry, and JUnit output.

## CI Integration

The BDD step in `.github/workflows/testing.yml`:

```yaml
- name: BDD Acceptance Tests
  run: go tool task test:specs:ci

- name: Upload BDD Report
  uses: actions/upload-artifact@v7
  if: always()
  with:
    name: specs-report
    path: specs-report.xml
```

`--flake-attempts=2` retries flaky specs once, reducing CI noise from transient container or network issues.

## Migration Roadmap

| Phase | Scope                                                                      | Status   |
| ----- | -------------------------------------------------------------------------- | -------- |
| 0     | Infrastructure: Ginkgo deps, SynchronizedBeforeSuite, CI, revive exemption | Complete |
| 1     | New modules: mandatory Ginkgo BDD spec; existing code untouched            | Ongoing  |
| 2     | Existing integration tests: migrate from testify/suite to Ginkgo           | Complete |
| 8     | Unit tests: testify table-driven retained permanently                      | Never    |

## Rules

1. **New modules must include BDD specs** in `tests/specs/<name>_spec_test.go`.
2. **Existing unit tests are never migrated** to Ginkgo. The testify table-driven pattern is the standard for unit tests.
3. **Use Labels** on every `Describe` container to enable targeted test execution.
4. **BeforeEach for setup, It for assertions**. Never assert in container nodes.
5. **Declare in container nodes, initialize in setup nodes** to prevent spec pollution.
6. **Per-process database isolation** is automatic. Do not hardcode database names in specs.
7. **Use `GinkgoWriter.Printf`** for debug output instead of `fmt.Println` or `t.Log`.
8. **DeferCleanup** in BeforeEach for per-spec cleanup; SynchronizedAfterSuite for suite-level.
9. **Build tag**: All files in `tests/specs/` must include `//go:build integration`.
10. **Use `sonic` for JSON**, never `encoding/json`.
11. **Use `types.Id()` for unique test values**, never hardcoded strings.
12. **Cleanup after each test** -- delete created database records in the test body.

## Environment Variables

| Variable                 | Default              | Description                                     |
| ------------------------ | -------------------- | ----------------------------------------------- |
| `SKIP_INTEGRATION_TESTS` | `false`              | Set to `true` to skip all integration/BDD tests |
| `POSTGRES_IMAGE`         | `postgres:16-alpine` | PostgreSQL container image                      |
| `REDIS_IMAGE`            | `redis:7-alpine`     | Redis container image                           |

## Tool Dependencies

The Ginkgo CLI is declared in `go.mod` via the `tool` directive:

```
tool (
    github.com/onsi/ginkgo/v2/ginkgo
)
```

This pins the Ginkgo CLI version to the project's module graph. Use `go tool ginkgo` to invoke it -- no separate `go install` required.

```
go tool ginkgo version
Ginkgo Version 2.28.1
```
