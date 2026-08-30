# Agent Note: Same-tab nav helper rejects javascript/data/vbscript hrefs

Status: implemented

## Problem

`flowbotIsPrimarySameTabNav` in `public/js/app.js` classifies a click as a same-tab document navigation so badge XHRs and chatagent EventSource can be aborted before the next GET. It skipped `javascript:` hrefs (those clicks do not load a document) but not `data:` or `vbscript:`, which browsers can also execute. CodeQL `js/incomplete-url-scheme-check` flags a `javascript:`-only prefix check as CWE-20 / CWE-184.

## Decision

Normalize the `href` with `decodeURI`, trim, and lowercase, then reject prefixes `javascript:`, `data:`, and `vbscript:` (and `#` fragments). A malformed percent-encoding falls back to the trimmed lowercase raw string. The helper still does not treat those clicks as navigation, so in-flight HTMX and observer sockets stay open. This is a denylist for non-document schemes, not an XSS sanitizer: ops-console hrefs are server-rendered. Socket teardown: [chatagent-sse-pagehide-teardown](2026-08-30-chatagent-sse-pagehide-teardown.md), [nav-badge-single-poller](2026-08-30-nav-badge-single-poller.md).

## Alternatives considered

- **Allowlist `http:` / `https:` plus relative paths.** Rejected: the helper's job is "will this click load a new same-tab document", which includes `mailto:`-unlike relative and `http(s)` links already in the shell. Expanding to an allowlist would change abort behavior for schemes the UI does not emit and is outside the CodeQL gap.
- **Add `data:` / `vbscript:` on the raw string only.** Rejected: CodeQL's documented check is case-insensitive and trimmed; `JavaScript:` and leading space would still look like navigation.
- **Dismiss the alert as a false positive.** Rejected: the same prefix is the classifier for aborting sockets; leaving `data:` through is the incomplete check the query names.

## Consequences

Clicks on `javascript:`, `data:`, and `vbscript:` hrefs (any case, optional surrounding space, percent-decoded) do not abort in-flight HTMX or close the approval EventSource. Ordinary same-tab `<a href="/...">` navigation is unchanged.

## Verification

CodeQL `js/incomplete-url-scheme-check` on `public/js/app.js` `flowbotIsPrimarySameTabNav`. There is no Go seam for this click classifier.
