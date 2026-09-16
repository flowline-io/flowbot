# Agent Note: Agent Skills name validation and skills-ref CI

Status: implemented

## Problem

Product skills under `docs/skills/` were authored to follow [agentskills.io](https://agentskills.io/specification), but import and form paths only required non-empty `name` / `description`. Invalid names (uppercase, consecutive hyphens, directory mismatch, oversized description) could enter the database via zip import. There was also no CI gate using the reference `skills-ref validate` tool.

## Decision

- Add `pkg/validate` (`SkillName`, `SkillDescription`, `SkillCompatibility`, `SkillDocument`, `ParseSkillMarkdown`, …) as the single home for Agent Skills identity rules (name, description length, optional compatibility length, name↔directory match), aligned with the published spec and with ASCII `a-z0-9-` naming used by Cap IDs and the Web UI.
- Enforce `SkillDocument` on SKILL.md import (`internal/server/chatagent`), `SkillName` / `SkillDescription` / `SkillCompatibility` on composer generation (`cmd/composer/action/skills`), and the same name/description checks on Web UI create/update.
- `go tool task skills` regenerates then runs full `skills:validate` (Go + `skills-ref@0.1.5`). CI workflow `.github/workflows/skills.yml` regenerates skills, fails on dirty `docs/skills`, and runs the same validate gate.
- Do not run `skills-ref` on Cursor-only skills (for example `.cursor/skills/flowbot-dev-loop`) that use non-spec frontmatter such as `disable-model-invocation`.

## Alternatives considered

- **Rely only on skills-ref (no Go package).** Rejected: server import and Web forms cannot depend on Node/npx at runtime; rules must live in-process.
- **Hand-roll validation only in chatagent.** Rejected: composer and Web would drift; Cap ID / skill name is a shared contract under `pkg/validate`.
- **Strict-reject unknown frontmatter fields on import (full skills-ref parity).** Rejected: imported Cursor/Claude skills may carry product extensions; identity rules are enough for product safety. Full field allowlisting stays a docs/skills CI concern via skills-ref.

## Consequences

- Invalid zip imports fail with clear agentskills identity errors (including oversized compatibility).
- Regenerating skills without committing output fails CI.
- Local `go tool task skills` requires `npx` for skills-ref.

## Verification

```bash
go test ./pkg/validate ./docs/skills ./internal/server/chatagent -run 'Skill|Import|ParseSkill' -count=1
go tool task skills
```
