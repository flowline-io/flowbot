# Agent Note: Firefly III B3 transaction CRUD and lists

Status: implemented

## Problem

`fireflyiii` only exposed create-transaction plus about/user/health. Agents could record expenses but could not list or correct transactions, or discover accounts/categories/budgets/bills/tags needed for accurate bookkeeping—unlike fuller CRUD peers such as karakeep.

## Decision

Ship a request-response B3 surface end-to-end (provider → capability → hub HTTP → client → CLI → skills):

- Transaction ops: `list_transactions`, `get_transaction`, `update_transaction`, `delete_transaction` (keep `create_transaction`)
- Read-only lists: `list_accounts`, `list_categories`, `list_budgets`, `list_bills`, `list_tags`
- Public transaction `id` is the Firefly **group** id; domain mapping uses the first split only
- Domain types stay thin (D1); create/update accept budget/bill/tags; update is partial via get→merge→put
- Lists use opaque page cursors; transaction list filters: type, start/end, account_id; accounts filter by type
- CLI: short verbs for the transaction primary resource (`list`/`get`/`update`/`delete`) plus resource-named list commands
- No webhook/events and no finance conformance in this change

## Alternatives considered

- Mirror the full Firefly REST surface: rejected; too large for a homelab capability and would turn Flowbot into a second Firefly client.
- Transaction CRUD only, without account/category/budget/bill/tag lists: rejected; create already depends on account/category names and B3 lists close the loop.
- Include events/conformance with the same change: deferred; orthogonal to CLI completeness and unstable until the domain contract settles.
- Expose journal/split ids as first-class update targets: rejected for v1; keep group-id parity with create output.

## Consequences

- Capability op count for fireflyiii is 13; hub `/service/fireflyiii` gains list/get/update/delete transaction routes and five resource list routes.
- Multi-split groups are not editable as separate lines; update merges only the first split but **re-sends all splits** on PUT so Firefly does not drop the rest. Callers that need per-split editing must wait for a later deepening.
- Partial update uses field presence (HTTP pointer body / CLI changed flags / client map). Empty strings and `tags: []` clear those fields; omitted keys leave them unchanged.
- Invalid Firefly tokens continue to fail via Accept JSON + empty-data checks (see [Accept JSON note](../bug-fix/2026-10-14-fireflyiii-accept-json-empty-about.md)).

## Verification

```bash
go test ./pkg/providers/fireflyiii/ ./pkg/capability/fireflyiii/ ./pkg/client/ ./cmd/cli/command/ ./internal/modules/hub/ -count=1
go tool task skills
```
