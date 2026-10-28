# Agent Note: Automate delete has its own scope

Status: implemented

## Problem

The automate HTTP API used one write gate for every non-GET method. `pipeline:run`, `function:run`, and `workflow:run` therefore authorized apply, run, and delete together. The MCP run preset adopted those scopes, so an MCP client token minted to run pipelines could also delete them. Listing, reading, and running are common; deleting a definition is not, and an accidental delete is not recoverable from the API.

## Decision

The automate groups split DELETE from the other writes in `auth.MinimumServiceScope`. DELETE under `automate/pipeline`, `automate/functions`, and `automate/workflow` requires `pipeline:delete`, `function:delete`, or `workflow:delete`. POST, PUT, and PATCH keep the run scopes, so apply and run are unchanged. `pipeline:run`, `function:run`, and `workflow:run` no longer satisfy a DELETE gate, and the delete scopes do not satisfy read. `HasMinimumServiceScope` keeps run-to-read fallback for GET and does not extend it to delete.

The scope names stay resource-oriented, matching the existing `pipeline:*` / `function:*` / `workflow:*` families, rather than one shared `automate:delete`.

`MCPRunScopes` stays read plus run. `MCPRunDeleteScopes` is run plus the three delete scopes. The web wizard exposes read, run, run-and-delete, and custom presets; the custom list already comes from `AllScopes`, so the three delete scopes appear there too. The run preset and its warning no longer claim delete.

Route scope denials return HTTP 403: the server error handler maps `protocol.ErrAccessDenied` to `StatusForbidden`, alongside the existing domain `ErrForbidden` mapping.

Only the delete authorization is split. Apply (YAML upsert) still uses the run scope, and MCP still exposes no apply or delete tool.

## Alternatives considered

- One `automate:delete` scope for all three groups — rejected. It breaks the established resource-oriented family and forces every token to choose between broad delete or none. The earlier merge decision also kept per-domain scopes ([note](../simplification/2026-09-05-merge-automate-modules.md)).
- Let `:run` also satisfy `:delete`, and add only a UI warning — rejected. That keeps the blast radius that motivated the split.
- Gate delete in the handler instead of the route — rejected. Authorization stays at the route gate, and handlers would run before rejection.
- Make the run preset include delete behind a checkbox that posts a partial scope list — rejected. A separate preset keeps the preselected scope list coherent and previewable on the same page.
- Return 400 for scope denials because the existing oops path did — rejected. 403 is the contract the docs and route tests already describe.

## Consequences

- Existing tokens with only a run scope cannot delete automation definitions. Mint the matching delete scope, or use `admin:*`.
- CLI `pipeline delete`, `workflow delete`, and function delete need the delete scope; apply and run keep working with the old run token.
- The MCP run token cannot delete through REST. The wizard's run-and-delete preset is the one-click way to grant both.
- Pipeline blueprint DELETE shares the `pipeline:delete` gate because it is a DELETE under `automate/pipeline`.

## Verification

- `go test ./pkg/auth/` covers DELETE mapping, run-does-not-delete, delete-scope acceptance, and the preset split.
- `go test ./internal/server/` covers `ErrAccessDenied` to 403.
- BDD `tests/specs/automate_delete_scope_spec_test.go` seeds a token from `auth.MCPRunScopes()` and expects 403 on all three delete paths.
- `docs/user-guide/mcp.md` carries the scope table; the token wizard labels and warning match it.
