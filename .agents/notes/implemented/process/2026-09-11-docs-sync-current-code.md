# Agent Note: Sync durable docs to 2026-09 code

Status: implemented

## Problem

Architecture README and PlantUML still described the pre-capability world (20 bot modules, `pkg/ability`, 17–18 providers, Fiber :8080 / MySQL-style ports). Developer-guide conformance and several Source lines still pointed at deleted `pkg/ability` paths. Getting-started CLI showed `workflow run` with a file path; self-hosting used map-shaped `modules:` while `module.Init` requires a list.

## Decision

Bring durable docs under `docs/` in line with current code (present tense, inventories only):

- `docs/architecture/README.md` + `architecture.puml` / `layers.puml` / `dataflow.puml` / `deployment.puml`: modules = automate/example/hub/life/web; providers = 29; packages = 37; CI = 10 workflows; entry points include gateway and agent; `capability.Invoke`; listen :6060; PostgreSQL 16 / :5432
- `docs/developer-guide/conformance.md` (+ README commands): `pkg/capability/...`
- Fix remaining `ability` strings in monitoring, notification-gateway Source, config-reference
- Getting-started: `workflow apply` then `workflow run <name>`; self-hosting modules as list entries
- database-reference categories for functions, web_account, agent, gateway, clips, expanded life
- docs/README and developer-guide link to root CONTRIBUTING / SECURITY / CoC
- user-guide README documents `/service/automate/{functions,pipeline,workflow}`
- agent architecture links `present_html` / htmlpreview; reference config comments DeepSeek `deepseek-flash`

Package comment in `pkg/capability/conformance` updated to say capability (not ability).

## Alternatives considered

- Regenerate only PlantUML and leave Markdown inventories — rejected; README counts drive onboarding more than diagrams
- Full rewrite of `docs/reference/schema.md` — rejected; it already declares lag; Ent schemas remain source of truth

## Consequences

- Contributors following architecture or conformance docs hit live paths
- Orphan map-shaped modules examples in third-party blogs remain wrong; self-hosting is the canonical shape

## Verification

- Inventories match `internal/modules/*`, `pkg/providers/*`, `pkg/*`, `.github/workflows/*`
- Grep of durable docs (excluding generated `docs/skills/`) for `pkg/ability` returns no hits
- `module.Init` still unmarshals modules as `[]json.RawMessage` (list)
