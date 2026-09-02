# Pipeline functions.invoke step UI

Status: implemented

## Problem

Pipeline editor step setup for `functions.invoke` showed `name` as a free-text field and rendered no inputs for `version` (`number`) or `event` (`any`) because `pipeline-editor.js` only mapped `int`/`int64` and `map[string]any` to form controls.

## Fix

- Extend param type handling for `number` and `any` in `public/js/pipeline-editor.js` and `pipeline_editor.templ`.
- Add `GET /service/web/pipelines/function-invoke-options` (published function names + version list) and a dedicated name/version selector block for `functions.invoke`, mirroring `agent-run-options`.
- `FunctionStore.ListPublishedVersionNumbersByNames` backs version lists in one query; list failures are logged and fall back to latest published version.
- Name/version support pipeline `{{expr}}` via text inputs; unknown saved values show as unavailable options; `event` (`any`) has a variable picker and accepts JSON or expressions.
- `capability.IntParam` accepts `uint64` integers from editor YAML (`goccy/go-yaml` decode) so `functions.invoke` `version` is not dropped at test/runtime.
- Step drawer snapshots into `drawerStep` and remounts via `drawerStepFormReady`. Dynamic `<select>` options are built in JS (`fillSelect` / `new Option`) because Alpine CSP `:value` on `x-for` options is applied too late, so the browser would show the first option (`core`, `a`).

## Verification

- `go test ./pkg/capability/... -run TestIntParam`
- `go test ./internal/store/... -run ListPublishedVersionNumbers`
- `go test ./internal/modules/web/... -run TestGetFunctionInvokeOptions`
- Manual: pipeline editor → `functions` / `invoke` → name/version dropdowns, `{x}` expressions, and event JSON textarea render and persist params.
