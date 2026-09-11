# Agent Note: Chat HTML artifacts with sandboxed preview

Status: implemented

## Problem

The chat agent can write HTML to the workspace and can publish markdown clips, but it has no way to show an interactive page in the session. Homelab dashboards, charts, and small tools need to run in the user's browser. Putting model-generated HTML into the main Web UI would inherit Flowbot cookies and the page CSP; the existing Docker/kern sandbox is a shell/code runtime, not a document preview.

## Decision

Interactive HTML is a first-class chat artifact via the product tool `present_html` (`internal/server/chatagent/tools/htmlpreview`).

- Upsert: `html` required; `title` and `id` optional (server generates `html_<hex>` when id is omitted). Caller-supplied id is `[A-Za-z0-9_-]` up to 64 characters so it is a stable DOM key. Same id replaces the previous version.
- Persistence is the session transcript only. The full document stays in the tool-call arguments; the tool result is a stub (`id`, `title`, `bytes`, `hash`). `transform.DefaultConvertToLLM` rewrites historical `present_html` `html` arguments to a placeholder so later turns and compaction do not reload the document.
- Cap is 256KB; overflow is a tool error, not truncation.
- Input is treated as a document: fragments are wrapped; CSP meta is inserted as the first `<head>` content (charset follows). Tags are not sanitized (scripts must remain).
- Web preview is an iframe with `sandbox="allow-scripts"` (never `allow-same-origin`) and `srcdoc`. Network lock is the injected CSP (`connect-src 'none'`, `default-src 'none'`, inline script/style only, `img-src data:`). Global `X-Frame-Options: DENY` is unchanged.
- UI: tool card plus Preview/Source tabs and an expand overlay. Only the latest completed card per artifact id mounts a live iframe and keeps Preview/Source; older cards stay in history as "updated" (title + note only).
- Registered on interactive session runs (Web / REST / DM / scheduled with a session). `NewSubagentRegistry` does not register it. Pipeline/ephemeral default tool sets omit it (`BaseToolNamesForRun`); explicit pipeline tools go through `SelectableSubagentTools`, which also excludes it. Default permission key `html` is allow; plan mode includes the tool. No new SSE event type; completed `tool` events may include `html`, `artifact_id`, and `call_id`.

## Alternatives considered

- **Workspace `.html` auto-preview / markdown fences.** Mixing source files or streaming fences with a product artifact has no stable id and no upsert.
- **Independent artifact table or HTML clip URL.** Extra source of truth; first period does not share HTML publicly.
- **Same-origin preview route with `X-Frame-Options: SAMEORIGIN`.** Stronger HTTP CSP, but punches a hole in the global DENY policy.
- **Docker sandbox as a mini HTTP server.** Wrong isolation primitive, slow, and needs ports and lifecycle.
- **JS plus CDN or vendor allowlist.** Useful later; first period requires inlined CSS/JS so the CSP can keep `connect-src 'none'`.

## Consequences

`present_html` is a product tool, not an engine coding tool. Compaction and dual-model conversion both go through `DefaultConvertToLLM`, so argument redaction lives there (tool name string `present_html`). The parent page CSP `default-src 'self'` continues to block iframe navigation to third-party origins.

Out of scope: public share URLs, workspace export, `get_html`, right-hand canvas, vendor script allowlists, Docker involvement.

## Verification

`go test ./internal/server/chatagent/tools/htmlpreview ./pkg/agent/transform ./pkg/agent/permission ./internal/server/chatagent ./pkg/views/partials ./internal/modules/web -count=1` covers wrap/CSP-first-in-head/size, argument redaction, default-allow permission, SSE `html` payload, history hydration, pipeline/subagent omission, and iframe `sandbox="allow-scripts"` without `allow-same-origin`. This is a product tool on the existing chat surface (no new HTTP route); unit tests are the owning layer. Checklist row W-11.
