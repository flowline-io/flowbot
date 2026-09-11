## Summary

<!-- What changed and why (1–3 sentences). Link related issues: Fixes #NNN -->

## Type of change

- [ ] Bug fix
- [ ] Feature / enhancement
- [ ] Refactor / simplification (no intended behavior change)
- [ ] Documentation
- [ ] Provider / capability
- [ ] UI / Web
- [ ] Tests / CI / tooling

## Checklist

- [ ] I read [CONTRIBUTING.md](https://github.com/flowline-io/flowbot/blob/master/CONTRIBUTING.md) and the nearest nested `AGENTS.md` for touched packages
- [ ] I did **not** hand-edit generated code (`*_templ.go`, ent, Swagger, `docs/skills/`); I regenerated via `go tool task` when needed
- [ ] Non-trivial change includes or updates an [Agent Note](https://github.com/flowline-io/flowbot/blob/master/.agents/notes/README.md) in this PR
- [ ] User-facing behavior: [CHANGELOG.md](https://github.com/flowline-io/flowbot/blob/master/CHANGELOG.md) updated under `## Unreleased` (or N/A)
- [ ] Tests updated when intended behavior changed; `go tool task lint` and relevant `go tool task test` / `go test ./...` pass locally
- [ ] BDD (`go tool task test:specs`) run if this change needs acceptance coverage and Docker is available (otherwise note skipped)

## Test plan

<!-- How reviewers can verify. Commands, UI paths, or N/A for docs-only. -->

-

## Notes for reviewers

<!-- Optional: risk areas, follow-ups, screenshots. -->
