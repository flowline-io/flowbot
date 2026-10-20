# Agent Note: Memos create and update send the Memo as the HTTP body

Status: implemented

## Problem

`flowbot memo create -c "test #work"` returned a memo name with an empty `Content`, and the note did not show up in the signed-in Memos account. `Get` on the same name confirmed the stored memo had no content (`Creator: users/1`, `Visibility: PRIVATE`).

Memos `CreateMemo` and `UpdateMemo` use gRPC-gateway `body: "memo"`. The HTTP JSON is the Memo resource. Flowbot posted `{"memo":{"content":"..."}}` and patched `{"memo":{...},"updateMask":[...]}`. The gateway ignores the unknown `memo` wrapper, so create stored a blank memo owned by the token user. A blank memo has no text and no tags, so it does not appear when looking for that note.

## Decision

`CreateMemo` posts the Memo fields at the top level. `UpdateMemo` patches the same shape and sends `updateMask` as a comma-separated query parameter. The nested `CreateMemoRequest` and `UpdateMemoRequest` types are removed so the wire shape cannot be wrapped again.

This applies to Memos 0.25 and 0.26; both map those RPCs with `body: "memo"`. Health-path fallback stays in [memos current-user path](2026-09-01-memos-current-user-path.md).

## Alternatives considered

- **Keep the wrapper and also send top-level fields.** Rejected: the gateway binds one body, and a duplicate shape would drift.
- **Treat empty create responses as errors without changing the request.** Rejected: the server already persisted a blank memo; the client has to send content where the gateway reads it.

## Consequences

- New creates store the text the caller passed, including tag syntax such as `#work`.
- Updates of content, visibility, and pinned status reach the field mask.
- Memos created before this change stay blank. Delete them and create again.

## Verification

- `go test ./pkg/providers/memos -run 'TestCreateMemo|TestUpdateMemo'` asserts the JSON body has top-level `content` and that `updateMask` is a query parameter.
