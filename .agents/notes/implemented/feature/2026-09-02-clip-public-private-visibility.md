# Agent Note: Clip public/private visibility

Status: implemented

## Problem

All clips shared the same access rule: anonymous visitors saw only title and description at `/c/:slug`, while any authenticated web user could read the full markdown. There was no per-clip control over whether content should be anonymously readable, and no way for operators to expose a clip for link sharing without changing global auth behavior.

## Decision

Add `is_public bool` (default `false`) on `clips`:

| Visibility | Anonymous `/c/:slug` | Authenticated web user |
|------------|------------------------|-------------------------|
| **private** (default) | HTTP 404 (indistinguishable from missing) | Full body + Copy MD |
| **public** | Full body + Copy MD | Full body + Copy MD |

- New clips are always created **private**; agents cannot set visibility via `clip_create` / `create_clip`.
- Any authenticated web user may toggle visibility from `/service/web/clips`.
- **private → public** requires the shared confirm modal; **public → private** is immediate.
- `clip_get` / internal capability reads ignore visibility (trusted path).
- Capability responses include read-only `is_public`.

## Alternatives considered

- **Gated preview for private clips (title + description for anonymous):** rejected — operators chose strict 404 to avoid leaking existence or preview text.
- **Agent-specified visibility at create:** rejected — visibility is an explicit human opt-in after review.
- **Creator-only toggle:** rejected — clips list already shows all clips to any web user; matching that permission model keeps v1 simple.

## Consequences

- Existing clips remain private after migration (`is_public` defaults to `false`).
- The prior anonymous gated UI (`clip-gated`, login CTA) and its i18n keys are removed from the reader page.
- Public clips are fully indexable/shareable without login.
- Visibility toggle failures use HTMX toasts (`toastErrorKey`) so the table row is not replaced with plain text.

## Verification

```bash
go tool task ent
go tool templ generate
go test ./internal/modules/web/... -run 'Clip'
go test ./internal/server/chatagent/tools/clip/...
go test ./pkg/capability/core/...
```

Manual: create clip via agent → list shows Private → Make public (confirm) → anonymous `/c/:slug` shows body; Make private → anonymous 404.
