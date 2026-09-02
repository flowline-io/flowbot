# Agent Note: Agent session detail export

Status: implemented

## Problem

The agent session detail page lists entries in the UI but offers no way to download the full persisted tree for offline debugging, archival, or sharing with maintainers.

## Decision

Add an **Export** button to the session detail page header. It links to `GET /service/web/agent-sessions/:id/export`, which reuses `chatagent.ExportSession()` and returns compact JSON (`SessionExport`) as `Content-Disposition: attachment` with filename `session-{id}.json`.

Web auth matches the detail page: `authenticateWeb()` only (no per-owner check; closed sessions allowed). Format matches the existing REST export at `/chatagent/sessions/:id/export`.

## Alternatives considered

- **Link to REST export.** Rejected: different auth (token scope + owner check) and inline JSON instead of download.
- **Pretty-printed JSON.** Rejected: larger files; compact matches REST and is easy to format locally.

## Consequences

- Ops users with web login can export any visible session, including closed ones.
- Export payload is entries + session metadata only (not plans/todos UI cards).

## Verification

- `TestAgentSessionExportAuthenticated` covers attachment headers, success body, 404, and closed session export.
- `TestAgentSessionDetailAuthenticated` asserts `data-testid="agent-session-export"` on the detail page.
