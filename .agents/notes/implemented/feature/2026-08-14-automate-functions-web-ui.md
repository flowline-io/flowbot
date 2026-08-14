# Agent Note: Automate Functions web UI

Status: implemented

## Problem

Named FaaS already has CLI and `/service/functions` REST (apply/list/get/export/delete/runs/call), but Automate nav had no Functions page. Operators could not draft/publish, inspect runs, or try a published version from the ops console without leaving the browser.

## Decision

`/service/web/functions` under Automate (nav + command palette) manages named functions in a Pipeline-shaped console:

- **ListAll** shows draft and published rows with status / unpublished-changes chips; list panel refreshes via HTMX `GET /functions/list` every 30s
- **Create** takes name + entrypoint; the server generates an HTTP token and writes a language-specific stub source (draft only)
- **Editor** is metadata form + single-file source (`main.py` | `main.sh` | `main.go`); Save draft / Publish use optimistic definition `version` (CONFLICT → toast + page reload)
- **Call URLs** on the editor show `POST /service/functions/call/{name}` and optional `/v/{n}` for ops copy (token stays out of the URL; auth via header/query)
- **Secrets** use `config.MaskedSecret` in reads; SaveDraft merges mask/empty to keep prior values
- **Try** invokes the latest published version via `capability.Invoke(hub.CapFunctions, invoke)` using `LatestPublishedVersion` (platform path; no function HTTP token)
- **Runs** tab lists run history; **Stats** tab loads per-function charts; Delete confirms then redirects to the list
- **List stats** load via HTMX (`GET /functions/stats`) with Pipeline-style summary cards + charts (success trend, duration, version pie); per-function `GET /functions/:name/stats` powers the editor Stats tab (no global summary cards). Aggregation reads `FunctionStore` (same pattern as pipeline/workflow stats); existence checks and lifecycle use `ActiveService()`
- No Web Export, no historical published-version browser, and no new `/service/functions` REST create/draft/publish endpoints (CLI `apply` stays draft+publish)

Domain surface: `pkg/functions` `Create`, `ListAll`, `GetDraft`, `SaveDraft`, `Publish`, `LatestPublishedVersion`; Catalog/store `ListAll`. Web handlers live in `internal/modules/web/function_webservice.go`.

## Alternatives considered

- Read-only ops console or apply-only UI — rejected; operators need draft/publish in-browser
- Bundle/raw YAML editor only — rejected; single entrypoint fits form + source pane
- Fake client-side draft — rejected; Catalog already has UpdateDraft/Publish
- List published-only — rejected; draft-only creates would disappear from the list
- Web Export of secrets — rejected; conflicts with redaction; keep CLI export

## Consequences

- Ops console is the primary lifecycle UI; CLI/REST remain for apply/export/call
- After first publish, “unpublished changes” is draft≠published field compare, not `status` alone
- Multi-tab edits rely on version conflict; Try is disabled until a published snapshot exists

## Verification

- `go test ./pkg/functions/ -run 'TestCreateListAll|TestSaveDraft'`
- `go test ./internal/modules/web/ -run 'TestFunctionWebCreateDraftPublishTry|TestFunctionWebserviceRulesRegistered'`
- `go test ./internal/store/ -run TestFunctionStats`
- Nav: Automate → Functions (`data-testid="nav-functions"`); command palette includes Functions → `/service/web/functions`
