# Agent Note: GitHub community onboarding files

Status: implemented

## Problem

The public repository scored about 62% on GitHub's community health checklist: README and LICENSE existed, but CONTRIBUTING, SECURITY, Code of Conduct, and a pull request template did not. Issue templates were generic and did not match how this project actually takes contributions (providers, docs, UI). Visitors had no clear path from star to first PR, and security reports had no documented private channel.

## Decision

Ship a minimal community pack aligned with existing standing orders rather than inventing a parallel process:

- Root `CONTRIBUTING.md` links `AGENTS.md`, `example/` packages, Agent Notes, lint/test commands, and generated-code rules.
- Root `SECURITY.md` points at GitHub Security Advisories and asks admins to enable private vulnerability reporting.
- Root `CODE_OF_CONDUCT.md` adopts Contributor Covenant 2.1; enforcement contact is the `flowline-io` org owners via GitHub (no public org email).
- `.github/PULL_REQUEST_TEMPLATE.md` encodes the same checklist contributors already need for review.
- Issue templates: bug, feature, new provider, documentation, UI; `config.yml` keeps blank issues enabled and adds contact links (docs, self-hosting, contributing, security).
- README gains a short Contributing section so the entry points are discoverable from the product page.
- PR and issue template Markdown links use absolute `github.com/flowline-io/flowbot` URLs so they resolve in the GitHub UI.

Labels referenced by templates (`bug`, `enhancement`, `documentation`, `provider`, `ui`) should exist on the repository; maintainers still apply `good first issue` / `help wanted` manually.

## Alternatives considered

- **Only README "Contributing" subsection** — rejected; GitHub community profile and contributor tooling expect the dedicated files.
- **Heavy governance (DCO, CLA, GOVERNANCE.md)** — rejected for pre-1.0 single-maintainer scale; GPL-3.0 contribution under CONTRIBUTING is enough.
- **Dedicated security@ email** — rejected until the org publishes one; Advisories are the durable channel.

## Consequences

- Community health checklist items for contributing, security, CoC, and PR template should pass after merge.
- Maintainers must enable **Private vulnerability reporting** in repository settings so the SECURITY.md and contact-link URLs work without friction.
- Create missing labels (`bug`, `enhancement`, `documentation`, `provider`, `ui`) once after merge if they are absent; apply `good first issue` / `help wanted` manually.
- CoC enforcement contact is org owners via GitHub until a dedicated public email exists.

## Verification

- Root files exist: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`.
- `.github/PULL_REQUEST_TEMPLATE.md` and five issue templates plus `config.yml` live under `.github/`.
- README Contributing section links the three root community files; CHANGELOG Unreleased cites this note.
