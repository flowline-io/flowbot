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
                         │  Integration Tests        │
                         │  testify/suite            │
                         │  tests/integration/       │
                         │  Requires Docker          │
                         ├───────────────────────────┤
                         │  Unit Tests               │
                         │  testify table-driven     │
                         │  pkg/** / *_test.go       │
                         │  No external deps         │
                         ├───────────────────────────┤
                         │  Fuzz Tests               │
                         │  testing.F                │
                         │  pkg/** / *_test.go       │
                         │  Retained permanently     │
                         └───────────────────────────┘
```

| Layer | Framework | Location | Migration |
|-------|-----------|----------|-----------|
| BDD Acceptance | Ginkgo + Gomega | `tests/specs/` | New modules must use, existing integration tests migrate gradually |
| Integration | testify/suite | `tests/integration/` | Phase out as specs cover the same ground |
| Unit | testify table-driven | `pkg/**/`, `internal/**/`, `cmd/**/` | **Never migrate** — retained permanently |
| Fuzz | `testing.F` | `pkg/**/` | **Never migrate** — Ginkgo does not support fuzzing |

## Directory Structure

```
tests/
├── specs/                              # Ginkgo BDD tests
│   ├── specs_suite_test.go             # TestMain + RunSpecs entry point
│   ├── lifecycle.go                    # SynchronizedBeforeSuite / AfterSuite
│   │                                   #   + per-process database isolation
│   ├── fixtures.go                     # HTTP request helpers
│   ├── health_spec_test.go             # Health check acceptance spec
│   ├── database_spec_test.go           # Database CRUD acceptance specs
│   └── bookmark_spec_test.go           # Module-level behavior specs
├── integration/                        # Legacy testify/suite integration tests
│   ├── suite_test.go
│   ├── health_test.go
│   ├── database_test.go
│   └── database_ext_test.go
└── fixtures/                           # Shared test data files
```

All files under `tests/specs/` use `//go:build integration` to prevent compilation during standard unit test runs.

## Infrastructure

### Parallel Database Isolation

Ginkgo's `--procs=N` flag runs N independent test processes. To prevent data conflicts, each process operates on an isolated database namespace using `GinkgoParallelProcess()`.

```
                     ┌──────────────────────┐
                     │  Process 1           │
                     │  SBS process1        │  Start PostgreSQL + Redis containers
                     │                      │  Serialize DSN → all processes
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
5. All processes run their assigned specs in parallel — zero data conflicts.
6. `SynchronizedAfterSuite` process 1 terminates containers after all processes complete.

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

| Node Type | Examples | Purpose |
|-----------|----------|---------|
| Container | `Describe`, `Context`, `When` | Organize specs hierarchically |
| Setup | `BeforeEach`, `AfterEach`, `JustBeforeEach`, `DeferCleanup` | Set up and tear down spec state |
| Subject | `It`, `Specify` | Make assertions about behavior |

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
                resp := JSONRequest("POST", "/service/bookmark/create",
                    []byte(`{"url":"https://example.com"}`))
                result, _ := App.Test(resp)

                Expect(result.StatusCode).To(Equal(200))
                Expect(ReadBody(result)).To(ContainSubstring("success"))
            })
        })

        Context("with an empty URL", func() {
            It("returns a validation error", func() {
                resp := JSONRequest("POST", "/service/bookmark/create",
                    []byte(`{"url":""}`))
                result, _ := App.Test(resp)

                Expect(result.StatusCode).To(Equal(400))
            })
        })
    })
})
```

### Labels

Labels enable selective test execution. Use them to categorize specs:

| Label | Convention | Usage |
|-------|-----------|-------|
| `smoke` | Fast, high-value tests | `ginkgo --label-filter="smoke"` |
| `module:<name>` | Tests for a specific module | `ginkgo --label-filter="module/bookmark"` |
| `integration` | Requires external services | `ginkgo --label-filter="integration"` |

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

| testify | Gomega |
|---------|--------|
| `require.NoError(t, err)` | `Expect(err).NotTo(HaveOccurred())` |
| `require.Equal(t, a, b)` | `Expect(b).To(Equal(a))` |
| `require.True(t, cond)` | `Expect(cond).To(BeTrue())` |
| `require.NotNil(t, v)` | `Expect(v).NotTo(BeNil())` |
| `assert.Equal(t, a, b)` | `Expect(b).To(Equal(a))` |
| `assert.Contains(t, s, sub)` | `Expect(s).To(ContainSubstring(sub))` |
| `s.Require().NoError(err)` | `Expect(err).NotTo(HaveOccurred())` |
| `s.Equal(a, b)` | `Expect(b).To(Equal(a))` |

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
# Standard go test works (no parallel within package)
go test -v -tags integration ./tests/specs/...

# gotestsum works
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
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: specs-report
    path: specs-report.xml
```

`--flake-attempts=2` retries flaky specs once, reducing CI noise from transient container or network issues.

## Migration Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| **0** | Infrastructure: Ginkgo deps, SynchronizedBeforeSuite, CI, revive exemption | Complete |
| **1** | New modules: mandatory Ginkgo BDD spec; existing code untouched | Ongoing |
| **2** | Existing integration tests: gradual migration from testify/suite to Ginkgo | Planned |
| **∞** | Unit tests: testify table-driven retained permanently | Never |
| **∞** | Fuzz tests: testing.F retained permanently | Never |

## Rules

1. **New modules must include BDD specs** in `tests/specs/modules/<name>_spec_test.go`.
2. **Existing unit tests are never migrated** to Ginkgo. The testify table-driven pattern is the standard for unit tests.
3. **Fuzz tests use `testing.F` exclusively**. Ginkgo does not support fuzzing.
4. **Use Labels** on every `Describe` container to enable targeted test execution.
5. **BeforeEach for setup, It for assertions**. Never assert in container nodes.
6. **Declare in container nodes, initialize in setup nodes** to prevent spec pollution.
7. **Per-process database isolation** is automatic. Do not hardcode database names in specs.
8. **Use `GinkgoWriter.Printf`** for debug output instead of `fmt.Println` or `t.Log`.
9. **DeferCleanup** in BeforeEach for per-spec cleanup; SynchronizedAfterSuite for suite-level.
10. **Build tag**: All files in `tests/specs/` must include `//go:build integration`.

## Mutation Testing Exclusion

Gremlins mutation testing is limited to unit test packages (`pkg/**`). The BDD specs directory is excluded because:

- Integration-level tests have high runtime overhead for mutation testing.
- Ginkgo's closure-based structure does not map cleanly to gremlins' AST-based analysis.
- The 60% threshold applies to the packages' own unit tests.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SKIP_INTEGRATION_TESTS` | `false` | Set to `true` to skip all integration/BDD tests |
| `POSTGRES_IMAGE` | `postgres:16-alpine` | PostgreSQL container image |
| `REDIS_IMAGE` | `redis:7-alpine` | Redis container image |

## Tool Dependencies

The Ginkgo CLI is declared in `go.mod` via the `tool` directive:

```
tool (
    github.com/onsi/ginkgo/v2/ginkgo
)
```

This pins the Ginkgo CLI version to the project's module graph. Use `go tool ginkgo` to invoke it — no separate `go install` required.

```
go tool ginkgo version
Ginkgo Version 2.28.1
```
