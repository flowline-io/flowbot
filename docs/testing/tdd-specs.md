# TDD Unit Test Standard

All unit tests follow Test-Driven Development with testify table-driven patterns. This document defines the development workflow, test structure, and quality standards.

## When unit tests are required

* Library / pure logic / behavior changes in Go packages: write or update table-driven unit tests (this doc).
* New modules or cross-boundary HTTP/event behavior: also add or update BDD specs — see [bdd-specs.md](./bdd-specs.md).
* Docs-only, AGENTS.md, or comment-only edits: unit tests are not required.
* Repo-wide policy summary: root [AGENTS.md](../../AGENTS.md) Testing policy.

## TDD Workflow

```
                ┌──────────┐
                │  Write a  │
                │  test     │
                └─────┬────┘
                      │
          ┌───────────▼───────────┐
          │  Run test → RED       │  Test fails — expected
          │  (no implementation)  │
          └───────────┬──────────┘
                      │
          ┌───────────▼───────────┐
          │  Write minimal code   │
          │  to pass the test     │
          └───────────┬──────────┘
                      │
          ┌───────────▼───────────┐
          │  Run test → GREEN     │  Test passes
          └───────────┬──────────┘
                      │
          ┌───────────▼───────────┐
          │  Refactor             │  Clean up, extract,
          │  (keep GREEN)         │  optimize — tests guard
          └───────────┬──────────┘
                      │
          ┌───────────▼───────────┐
          │  Run all tests        │  Full suite must stay GREEN
          └───────────────────────┘
```

### Red Phase

1. Write exactly one test that describes the behavior you want.
2. Name the test after the function and scenario: `TestParseAction_ValidInput`.
3. Run the test to confirm it fails for the expected reason (not a compile error, not nil pointer).
4. Do not write implementation code during this phase.

### Green Phase

1. Write the **minimum** code to make the test pass.
2. Do not add features, error handling, or optimizations the test does not demand.
3. Run only the failing test to verify it passes, then the full suite.

### Refactor Phase

1. Remove duplication in both test and production code.
2. Extract helpers, constants, interfaces where appropriate.
3. Run the full test suite after each refactoring step.
4. Do not change behavior — tests stay GREEN throughout.

## Development Order

For each new function or change:

1. **Start with the test file**: Create `<name>_test.go` before writing implementation.
2. **Happy path first**: Write the primary success case as the first table entry.
3. **Error cases second**: nil input, invalid input, boundary conditions, type mismatches.
4. **Edge cases third**: empty values, zero values, maximum/minimum limits, concurrency.
5. **At least 3 cases per table**: Every table must contain a minimum of 3 test cases. Single-case tests still wrap in `t.Run`.
6. **Write the implementation**: Only after the test structure is defined.

## Test File Organization

Tests are co-located with source code using Go's `*_test.go` convention:

```
pkg/workflow/
├── workflow.go           # Implementation
├── workflow_test.go      # Tests for workflow.go
├── types.go
└── types_test.go
```

Tests use the same package name with `_test` suffix to access only the public API:

```go
package workflow_test  // separate package — tests the public interface
```

When testing unexported internals is necessary (rare), use the same package:

```go
package workflow  // same package — access to unexported symbols
```

Prefer the `_test` suffix package unless testing internal logic that cannot be exercised through the public API.

## Standard Pattern

```go
func TestFunctionName_Scenario(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
        wantErr error
    }{
        {
            name:    "happy path",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "edge case empty input",
            input:   emptyInput,
            want:    zeroValue,
            wantErr: false,
        },
        {
            name:    "error invalid input",
            input:   invalidInput,
            wantErr: true,
            wantErr: types.ErrInvalidArgument,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got, err := FunctionUnderTest(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErr != nil {
                    assert.ErrorIs(t, err, tt.wantErr)
                }
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Variant: Multi-Input Functions

When the function takes multiple inputs, describe each in the struct:

```go
tests := []struct {
    name    string
    action  string
    wantType string
    wantErr bool
}{
    {name: "valid capability action", action: "capability:bookmark.list", wantType: "capability"},
    {name: "docker action", action: "docker:nginx:latest", wantType: "docker"},
    {name: "empty action", action: "", wantErr: true},
}
```

### Variant: Setup-Heavy Tests

When each case requires different setup that cannot be expressed in a struct, use inline subtests:

```go
func TestFindByEvent(t *testing.T) {
    t.Parallel()

    t.Run("method-match", func(t *testing.T) {
        t.Parallel()
        d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
        matched := d.FindByEvent("bookmark.created")
        require.Len(t, matched, 1)
    })

    t.Run("method-no-match", func(t *testing.T) {
        t.Parallel()
        d := Definition{Name: "p", Trigger: Trigger{Event: "bookmark.created"}}
        matched := d.FindByEvent("other.event")
        require.Empty(t, matched)
    })
}
```

## Assertion Rules

| Function              | Behavior                                             | Import                                |
| --------------------- | ---------------------------------------------------- | ------------------------------------- |
| `require.Xxx(t, ...)` | **Fatal** — stops the subtest immediately on failure | `github.com/stretchr/testify/require` |
| `assert.Xxx(t, ...)`  | **Non-fatal** — records failure, continues execution | `github.com/stretchr/testify/assert`  |

### Usage Guidelines

- **`require` for preconditions**: Use when subsequent assertions depend on the result (e.g., `require.NoError` before accessing the returned value).
- **`require` for setup**: Validating test setup (e.g., `require.True(t, db.Ping() != nil)`).
- **`assert` for behavior**: Use when multiple independent assertions should all be checked (e.g., `assert.Equal` on struct fields).
- **`assert` for non-critical checks**: Use when the test can meaningfully continue after a failure.

```go
// require for precondition — accessing got[0] depends on len check
require.Len(t, got, 1)
require.NoError(t, err)

// assert for behavioral checks — both should be checked
assert.Equal(t, "expected-name", got[0].Name)
assert.True(t, got[0].Enabled)

// Error checking pattern
if tt.wantErr {
    require.Error(t, err)              // fatal — no point continuing
    assert.ErrorIs(t, err, tt.wantErr) // non-fatal — extra detail
    return
}
require.NoError(t, err) // fatal — should never error
```

## Parallel Execution

All tests declare `t.Parallel()` at both the top level and within each `t.Run` subtest:

```go
func TestFoo(t *testing.T) {
    t.Parallel()          // top-level

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // subtest
            // ...
        })
    }
}
```

This maximizes concurrency. Tests run in `t.Parallel()` are paused until the parent function returns, then scheduled together.

### Parallel Constraints

- Tests using global mutable state (e.g., `os.Setenv`) should use `t.Setenv` or proper cleanup.
- Tests sharing a database fixture should not use `t.Parallel` against the same data.
- Read-only globals (e.g., compiled regexps, template engines) are safe for parallel access.
- Integration tests using shared containers do not use `t.Parallel()`.
- Never remove `t.Parallel()` to hide race conditions — fix shared-state serialization or other root causes instead.

## Test Naming

| Pattern                     | Example                      | Usage                                 |
| --------------------------- | ---------------------------- | ------------------------------------- |
| `Test<Function>_<Category>` | `TestParseAction_ValidInput` | Primary naming convention             |
| `Test<Function>`            | `TestValidateDAG`            | When only one category, with subtests |
| `Test<Struct>_<Method>`     | `TestRenderContext_Render`   | Methods on types                      |

Table entry names use descriptive `snake_case`:

```go
{name: "happy path with valid data"}
{name: "error when input is nil"}
{name: "edge case zero length slice"}
```

## Error Assertion Patterns

```go
// Basic error check
require.NoError(t, err)

// Error expected, type irrelevant
require.Error(t, err)

// Exact error match (errors.Is)
assert.ErrorIs(t, err, types.ErrNotFound)

// Error expected but not nil
require.Error(t, err)

// Error string contains
assert.Contains(t, err.Error(), "connection refused")
```

## Mock and Stub Policy

The project does not use `testify/mock`. Alternatives:

- **Pure functions**: Test with real inputs and outputs. Prefer this approach.
- **Interface-based stubs**: Define a local interface and pass a stub implementation in tests.
- **Table-driven**: Use the anonymous struct to provide different dependency behaviors via fields.

```go
// Prefer: test the real function with varied inputs
tests := []struct {
    name    string
    reader  io.Reader    // real or stub via strings.NewReader / bytes.Buffer
    want    string
}{...}

// Avoid: mock frameworks
// var mockClient = new(MockClient)  // not used in this project
```

## Benchmark Tests

Benchmark functions use `func Benchmark*(b *testing.B)`:

```go
func BenchmarkParseAction(b *testing.B) {
    for b.Loop() {
        ParseAction("capability:bookmark.list")
    }
}
```

Use `b.Loop()` instead of `for i := 0; i < b.N; i++` (Go 1.24+).

## Race Testing

Run race detection locally and in CI:

```bash
go test -race ./...                # single pass
go tool task test:race            # 10 iterations
```

Tests that are inherently racy (by design) should be documented and excluded via build tags.

## Short Mode

Integration and long-running tests should respect `testing.Short()`:

```go
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long-running test in short mode")
    }
    // ...
}
```

`go test -short ./...` skips these tests. Use `go tool task test:short` in development loops.

## Coverage

Generate coverage reports to identify untested paths:

```bash
go tool task test:coverage
```

Coverage alone is not a sufficient quality metric. Combine with mutation scores and BDD acceptance coverage.

## When NOT to Use Table-Driven

- **Integration tests** using `testify/suite.Suite` with container setup/teardown.
- **BDD acceptance tests** using Ginkgo `Describe`/`Context`/`It` (see `bdd-specs.md`).
- **Tests requiring per-case setup** that cannot be expressed in a struct.
- **Generated test files** (Ent, Swagger codegen output).

## Code Review Checklist

- [ ] Test file is co-located with source: `<name>_test.go`.
- [ ] Test function named `Test<Function>_<Category>`.
- [ ] Uses `t.Parallel()` at top-level and in each `t.Run`.
- [ ] Table has at least 3 cases with descriptive `name` fields.
- [ ] Happy path is the first entry.
- [ ] Error paths are covered (nil input, invalid input, boundary conditions).
- [ ] `require` for fatal assertions, `assert` for non-fatal.
- [ ] No panics in test code (use `require` to guard).
- [ ] No use of `encoding/json` — use `github.com/bytedance/sonic`.
- [ ] `testing.Short()` respected for long-running tests.
