# Agent Note: Deploy command uses default notify template

Status: implemented

## Problem

The hub `deploy` command notified through template ID `github.deployment`. That ID is listed as a predefined template but is not seeded, so a live deploy fails with `template github.deployment not found` after the Drone build is already created.

## Decision

`deploy` sends through `notify.GatewaySendDefaults` with a `summary` payload (title plus optional Drone base URL). It uses the operator-configured default template and default channel. Missing defaults remain a soft skip via `WarnSkipNoDefault`.

## Alternatives considered

- **Seed a `github.deployment` template.** Rejected: one more named template to maintain for a single command; operators already configure a default template for ad-hoc alerts.
- **Keep `GatewaySendDefaultChannel` with a dedicated ID.** Same operational cost as seeding; still fails on installs that never created the row.

## Consequences

- Deploy no longer requires a `github.deployment` row in `notify_templates`.
- Message body follows the default template (`{{ .summary }}` when that is the configured body).
- The user-guide predefined-template list does not include `github.deployment`.

## Verification

`go test ./internal/modules/hub -run TestDeployNotifyPayload -count=1` covers payload fields. `go tool task lint`.
