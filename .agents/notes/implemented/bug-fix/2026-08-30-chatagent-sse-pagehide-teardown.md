# Agent Note: Close chatagent observer SSE on pagehide

Status: implemented

## Problem

Thread UI (`Inspect entries`) and entry detail (`Back to session`) each open an `/events` EventSource. Switching those pages left the previous stream open: `initApproval` never closed on `pagehide`, and the error handler rescheduled reconnect after unload. After a handful of round-trips the browser HTTP/1.1 per-host socket pool filled and the whole web UI stopped loading.

## Decision

Tear down the observer in `public/js/chatagent-approval.js`: close the EventSource and cancel reconnect on `pagehide` and on primary same-tab `<a href>` clicks (bubble phase, shared `flowbotIsPrimarySameTabNav` in `app.js`, so the next document GET can take the socket). Skip reconnect while stopped, replace a second `initApproval` on the same panel, and reconnect only on a bfcache `pageshow`. Navbar badge fan-out: [nav-badge-single-poller](2026-08-30-nav-badge-single-poller.md). Href schemes the helper does not treat as navigation: [nav-href-scheme-denylist](2026-08-30-nav-href-scheme-denylist.md).

## Alternatives considered

- **Rely on document unload to drop EventSource.** Rejected: bfcache and the error-handler `setTimeout(connect)` keep sockets after the user has left the page.
- **Disable live `/events` on the inspect page.** Rejected: inspect still needs approval prompts; the leak was lifecycle, not the observer itself.

## Consequences

- Inspect ↔ thread round-trips keep at most one live observer socket.
- bfcache restore reconnects; a normal navigation does not.

## Verification

- `public/js/chatagent-approval.js` `initApproval` closes the EventSource on `pagehide` and primary same-tab link clicks (`flowbotIsPrimarySameTabNav` in `app.js`), ignores error-reconnect while stopped, destroys a previous observer on the same panel, and reconnects only on a persisted `pageshow`. There is no Go seam for this browser lifecycle.
