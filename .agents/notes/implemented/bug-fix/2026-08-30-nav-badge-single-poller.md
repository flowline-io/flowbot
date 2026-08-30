# Agent Note: Single navbar badge poller per endpoint

Status: implemented

## Problem

The ops shell fired `hx-get` for `/inbox-badge` twice and `/approval-badge` three times on every page (`load, every 20s`), while chatagent thread/inspect also held an `/events` EventSource. Chrome HTTP/1.1 allows about six connections per host, so inspect ↔ thread navigation queued the next HTML document behind pending badge polls — Network shows those GETs stuck `(pending)` and the UI freezes.

## Decision

Keep one HTMX poller per badge URL (`nav-inbox-badge`, `nav-agent-approval-badge`) with `hx-sync="this:abort"`. Replica slots (`nav-mobile-*`, `nav-agents-approval-badge`) receive the same chip via `hx-swap-oob="innerHTML:#id"`. Free in-flight badge XHRs on primary same-tab navigation (bubble click, after `data-confirm` may `preventDefault`) and on `pagehide`. Observer SSE teardown lives in [chatagent-sse-pagehide-teardown](2026-08-30-chatagent-sse-pagehide-teardown.md). Href schemes the nav helper does not treat as navigation: [nav-href-scheme-denylist](2026-08-30-nav-href-scheme-denylist.md).

## Alternatives considered

- **Leave duplicate pollers; only close EventSource.** Rejected: the Network trace still showed stacked `inbox-badge` / `approval-badge` plus the next page GET pending.
- **One combined `/nav-badges` endpoint.** Rejected: inbox and approval already have handlers and tests; OOB copies keep those contracts.

## Consequences

- A thread page at rest uses one EventSource plus two badge polls (and session-badge on load), leaving connection slots for navigation.
- Mobile and Agents-menu chips still update; they are no longer independent SSE-competing requests.

## Verification

- `go test ./pkg/views/layout/ -run TestBaseLayout/nav_badges_poll_each_endpoint_once` — one `hx-get` each for inbox-badge, approval-badge, and session-badge; replica slot ids present; two `hx-sync="this:abort"` pollers.
- `go test ./pkg/views/partials/ -run TestNavBadgeFragmentsCopyToReplicaSlots` — OOB selectors for mobile/menu replicas.
- `go test ./internal/modules/web/ -run 'TestApprovalBadge|TestInboxBadge'` — badge responses include replica OOB.
