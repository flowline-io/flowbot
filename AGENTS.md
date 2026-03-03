# Agents Guide for Flowbot Repository

This document is targeted at automated coding agents (including Copilot-style assistants) that operate
within the `flowbot` codebase. It collects the common commands, style rules and conventions that
humans currently follow so that agents can behave consistently and safely.

---

## 🛠️ Build, Lint & Test Commands

The project uses [Taskfile](https://taskfile.dev) as a thin front‑end for build commands. All of the
example commands below may be executed from the repository root (`d:\Projects\flowbot` on Windows).

### Running the Full Suite

```bash
# run everything: tidy → swagger → format → lint → test → statistics
task default

# quick interactive runs
task build           # compile main server binary
task build:agent     # compile agent
task build:app       # compile admin PWA + server
task build:composer  # composer CLI tool

# code quality
task lint            # Go lint (revive) + actionlint
task format          # go fmt + prettier
task tidy            # go mod tidy

# security
task secure          # govulncheck
task leak            # gitleaks
task gosec           # gosec analysis

# tests
task test            # all unit tests via gotestsum
task test:all        # alias of task test (run on GH actions)
task test:coverage   # run tests with coverage report

# code generation / docs
task swagger         # regenerate Swagger via swag
task dao             # generate DAO from DB schema

# other utilities
task cloc            # line counts (cloc)
task scc             # code statistics (scc)
```

### Running a Single Test

Agents should prefer `task` wrappers but may also invoke `go test` directly when targeting a
specific package or symbol. Examples:

```bash
# run all tests in a package
go test ./pkg/utils

# run a specific test function by name
go test ./internal/bots/agent -run ^TestAgentStart$

# use gotestsum with filtering
go tool gotestsum --packages="./pkg/utils/..." -- -run=TestFoo
```

`gotestsum` is installed as a Go tool and is the default runner for `task test`.

> **Note**: When writing new tests, put them next to the implementation in `*_test.go` files and
> follow the existing naming patterns (`Test` prefix, table‑driven tests etc.).

### Executables & Binaries

- `bin/flowbot` – main server
- `bin/flowbot-agent` – background agent daemon
- `bin/flowbot-app` – admin PWA server
- `bin/composer` – CLI code‑generator/migrator

---

## 📐 Code Style Guidelines

Flowbot is a pure Go project with a small amount of frontend markup (WASM, HTML, YAML) and GitHub
workflows. The style rules below are enforced by formatting tools and lint configurations; agents
should abide by them automatically when producing or modifying code.

### Formatting & Imports

1. **`go fmt`** is the canonical formatter. Run it before committing or use `task format`.
2. Use `goimports` or the editor integration to maintain grouped imports: standard libraries first,
   blank line, third‑party packages, blank line, internal packages.
3. Avoid unused imports – the compiler will reject them; `goimports` also removes them.
4. Run `npx prettier --write .` for non‑Go files (Markdown, JSON, YAML, JS) to keep the repo
   consistent.

### Linting

- The repository uses [revive](https://github.com/mgechev/revive) with a custom `revive.toml`. The
  configuration is strict (`severity = "error"`) and many rules are enabled; see the file for
  specifics. In particular:
  - `blank-imports`, `dot-imports`, `error-naming`, `import-shadowing`, `unused-parameter`, etc. are
    active.
  - A few rules are disabled to suit project patterns (e.g. `exported`, `var-naming`).
- GitHub Actions workflows run `revive` on every push (`.github/workflows/build.yml`).
- Workflow YAML files are validated with [`actionlint`](https://github.com/rhysd/actionlint) as part
  of `task lint:action`.

Agents generating or modifying code must ensure it passes `revive` or the CI will fail. They may
lint incrementally by running the same command used in `task lint`.

### Naming Conventions

- **Packages** are all lowercase, short and often describe a functional area (`rdb`, `executor`,
  `notify`, `bots`). Avoid underscores or mixed case.
- **Files** are lowercase with underscores when necessary (`shell_test.go`, `list.go`). Test files
  use `_test` suffix and live in the same directory as the implementation.
- **Types** and **functions** use CamelCase; exported symbols start with a capital letter, unexported
  start with a lowercase letter.
- **Constants** use `CamelCase` or `ALL_CAPS` only when representing true constants; otherwise follow
  normal naming. Error variables typically begin with `err` (e.g. `var errNameEmpty = errors.New(...)`).

### Error Handling

- Always check `err != nil` immediately. Wrap errors with `%w` when propagating:
  `return fmt.Errorf("failed to do X: %w", err)`.
- Use the standard `errors` package for simple errors (`errors.New`). Custom sentinel errors are
  declared as package‑level `var errFoo = errors.New("...")`.
- Do not ignore returned errors; if a value is unused it must be assigned to `_` or the call must
  be removed.
- Logging functions (e.g. `flog.Error`, `flog.Debug`, etc.) accept an `error` as argument. Log the
  error where appropriate but still return it unless the caller can safely swallow it.

### Error Constants & Wrapping

- Define sentinel errors at the top of the file when they are checked by callers. Use `fmt.Errorf`
  with `%w` to wrap when adding context.
- Avoid `panic` except during initialization or when an invariant is truly violated. Tests may
  rely on `t.Fatal`/`t.Fatalf` when an unexpected condition occurs.

### Comments and Documentation

- Public (exported) types, functions, constants and variables should have a doc comment starting
  with the name of the symbol (`// Foo does ...`). Revive enforces some rules but it may be
  disabled for generated code.
- Package comments belong in a `doc.go` file at the package root.
- A line length of ~100 characters is the unwritten convention, but the formatter will not enforce
  it; break long strings or comments manually.
- FIXME/TODO comments are allowed sparingly; prefix with `TODO(<name>):` or `FIXME:`.

### Tests and Coverage

- Unit tests live alongside the code and use `testing.T` and `require`/`assert` from `github.com/stretchr/testify`.
- Table-driven tests and subtests are common; use `t.Run` where appropriate.
- Use `go tool gotestsum` for better output on CI but regular `go test` is fine locally.
- Benchmark tests use `func BenchmarkXxx(b *testing.B)` and are run with `go test -bench .` if
  needed.

### Generated Code

- The repository contains some generated files (DAOs, Swagger docs). Generated files include a
  header comment; revive is configured to ignore them by default (`ignoreGeneratedHeader = false`).
- When regenerating, run the appropriate `task` (e.g. `task swagger`) and commit the results.

### GitHub & CI

- Keep the build badge (in `README.md`) up to date by ensuring workflow files are passing.
- Commits should be signed-off if a DCO is required (not enforced by agents).
- Pull requests trigger `build.yml` which runs lint/test/secure; failing CI blocks merging.

---

## 🧰 Project Structure Patterns

The codebase is modular and layered. Agents should observe existing directory layouts to place
new code:

```
cmd/          # entry point binaries (server, agent, composer, app)
internal/     # implementation of bot modules, platform adapters, etc.
pkg/          # reusable libraries and utilities
app/          # webapp sources for the admin PWA
config/       # configuration samples and loaders
docs/         # generated and hand-written documentation
```

- `internal/` packages are not importable by outside modules; use `pkg/` for any package that may be
  shared by multiple components or by external consumers.
- Database access is encapsulated in `internal/store` and `pkg/rdb`; use the former for bot logic
  and the latter for library‑style utilities.

### Dependency Management

- Go modules are used (`go.mod` at repo root). Add dependencies with `go get` or let the build
  add them automatically, then run `go mod tidy`.
- Avoid pinning to `replace` unless absolutely necessary; follow upstream updates using
  `go get -u` and review `go.mod` changes in PRs.

### Configuration Files

- YAML files under `config/` and `docs/config` provide examples. The runtime expects
  `flowbot.yaml` at the root (copied from `docs/config/config.yaml`).
- Agent and server read configuration via the `pkg/config` package which uses `github.com/spf13/viper`.

### Windows Specifics

- Paths in documentation use Unix style but Windows terminals work with forward slashes in Go
  commands. Taskfile commands are platform‑agnostic.

---

## 📎 Additional Notes

- No custom `.cursorrules` or Copilot instructions exist in this repository. Agents can follow the
  general Go conventions documented above.
- The `revive.toml` file lives in the root and contains all linting configuration; update it if new
  rules are needed.
- Some packages include `// revive:disable` comments inline to silence specific warnings; mimic
  that style when an exception is required.
- For frontend work (WASM), the build uses the Go toolchain with `GOOS=js GOARCH=wasm` and the
  `web/` directory holds static assets. Prettier formats HTML/JS.

---

Having a complete AGENTS.md allows any automated assistant to quickly understand how to build,
verify, and extend Flowbot while staying within the project's conventions. Agents should refer to
this file before producing significant changes and may update it if new patterns emerge.
