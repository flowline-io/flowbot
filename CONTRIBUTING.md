# Contributing to Flowbot

Thanks for helping improve Flowbot. This guide is for humans and agents opening issues or pull requests.

## Before you start

1. Read the product overview in [README.md](README.md).
2. Skim standing orders in [AGENTS.md](AGENTS.md) (boundaries, verification, English-only code/docs).
3. For package work, read the nearest nested `AGENTS.md`. When touching providers, capabilities, or modules, also open the matching `example/` package:
   - [pkg/providers/example](pkg/providers/example)
   - [pkg/capability/example](pkg/capability/example)
   - [internal/modules/example](internal/modules/example)

Security issues: do **not** open a public issue. Follow [SECURITY.md](SECURITY.md).

Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Ways to contribute

Open a structured issue from [New issue](https://github.com/flowline-io/flowbot/issues/new/choose):

| Kind | Template |
| --- | --- |
| Bug fix | Bug report (or a focused PR with reproduction notes) |
| Feature / enhancement | Feature request first for non-trivial ideas |
| New provider / capability | New provider / capability; follow the `example/` packages |
| Docs | Documentation |
| UI / Web | UI improvement |

Issues labeled `good first issue` or `help wanted` are the best entry points for new contributors.

Until 1.0, prefer correct foundations over compatibility shims. Domain event names stay stable; see the pre-release stance in [AGENTS.md](AGENTS.md).

## Development setup

Requirements: Go 1.26.6+, PostgreSQL, Redis, [Task](https://taskfile.dev) (`go tool task`), Docker for BDD specs.

```bash
git clone https://github.com/flowline-io/flowbot.git
cd flowbot
cp docs/reference/config.yaml flowbot.yaml
# Edit flowbot.yaml — set postgres.dsn and redis.url
go tool task build
go tool task run
```

More detail: [docs/getting-started/README.md](docs/getting-started/README.md), [docs/self-hosting.md](docs/self-hosting.md), [docs/developer-guide/README.md](docs/developer-guide/README.md).

## Pull request checklist

1. Prefer small, focused PRs. Extend existing files unless you are adding a new logical component.
2. Do **not** edit generated code (ent, templ `*_templ.go`, Swagger artifacts, `docs/skills/`). Change generators or sources, then run the matching `go tool task` target (`ent`, `templ`, `swagger`, `skills`).
3. Modules never import `pkg/providers/*` — call `capability.Invoke`. Database queries stay in `internal/store`. Use `github.com/bytedance/sonic` instead of `encoding/json` Marshal/Unmarshal.
4. Non-trivial changes include an [Agent Note](.agents/notes/README.md) in the same PR (`feature`, `bug-fix`, `simplification`, `architecture`, `process`, or `testing`).
5. Update [CHANGELOG.md](CHANGELOG.md) under `## Unreleased` when user-facing behavior changes.
6. Tests describe intended behavior. When behavior changes, update tests in the same PR and say why ([testing policy](docs/testing/README.md)).
7. Before requesting review:

```bash
go tool task lint
go tool task test          # or package-scoped: go test ./path/to/package/...
# Optional if you have Docker and the change needs acceptance coverage:
go tool task test:specs
```

8. Fill out the pull request template. Link related issues.

## Code style (summary)

Full standing orders live in [AGENTS.md](AGENTS.md). Lint owns import order, naming, and JS quotes (`go tool task lint`, `revive.toml`, oxlint) — do not bikeshed those in review.

- English for comments, docs, commit messages, and code identifiers.
- Exported symbols need godoc; unexported symbols need comments only for non-obvious *why*.
- No organizational comments that restate the code; no emojis in code or commit subjects.
- Wrap errors with `%w`; use `types.ErrNotFound` / `ErrForbidden` / `ErrProvider` where appropriate.

## Documentation

- One home per fact: [docs/AGENTS.md](docs/AGENTS.md). Link instead of restating rules.
- User/operator docs: `docs/user-guide/`, `docs/getting-started/`, `docs/self-hosting.md`.
- Website content: `docs/website/` (English-first; see [docs/website/NORTH_STAR.md](docs/website/NORTH_STAR.md)).

## License

By contributing, you agree that your contributions are licensed under the [GNU GPL v3.0](LICENSE).
