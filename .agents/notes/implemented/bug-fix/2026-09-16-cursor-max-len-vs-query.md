# Agent Note: Opaque pagination cursors use CursorMaxLen

Status: implemented

## Problem

CLI bookmark pagination failed on page two with `cursor exceeds maximum length of 100`. The server returns HMAC-signed opaque cursors from `capability.EncodeCursor` (typically 200–300+ characters). The Flowbot client validated those cursors with `validate.QueryMaxLen` (100), which is meant for short search text, so any real next_cursor was rejected before the HTTP call.

## Decision

Introduce `validate.CursorMaxLen` (4096) for opaque pagination cursors. Bookmark list/search and Trilium list validators use `CursorMaxLen`; search query strings keep `QueryMaxLen`. Memo list already allowed 4096 and now shares the named constant.

## Alternatives considered

- **Raise QueryMaxLen to 4096** — rejected; that would also loosen free-text search / query bounds unrelated to pagination.
- **Shorten EncodeCursor tokens** — rejected; signature and provider cursor payload need the current size; compressing for a misapplied client limit is the wrong seam.
- **Drop client-side cursor length checks** — rejected; keep a generous ceiling against accidental huge query strings.

## Consequences

- Operators can page past the first `limit` of bookmarks (and notes) via CLI.
- Callers that still cap cursors at 100 must switch to `CursorMaxLen`.
- Server/adapters already accept long cursors; this was a client-only rejection.

## Verification

`go test ./pkg/client -run 'TestValidate(ListBookmarks|SearchBookmarks|ListNotes)Query'` asserts cursors longer than `QueryMaxLen` pass and `CursorMaxLen+1` fails.
