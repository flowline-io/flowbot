# Agent Note: Manual retry of a failed pipeline run

Status: implemented

## Problem

A failed pipeline run (for example Kanboard `create_task` on run 73) could not be salvaged from the UI. Event consumption is already recorded, so firing the trigger again is a no-op. Operators had to mint a new run or edit YAML, which either skipped remaining work or re-executed successful mutation steps.

## Decision

Failed-run **manual retry** lives on the run detail page (`POST /service/web/pipelines/:name/runs/:runID/retry`).

- Same run ID: `failed → start` in place. Do not create a new run.
- Continue from the first failed step. Successful steps are not re-invoked; their stored `result` is replayed into the render context. If a published YAML insert sits before a Done step, that insert runs, but the Done step is still skipped (not a second row).
- Re-render params from the **current published** definition plus the original `data_events` row. No stored snapshot, no draft YAML.
- Failed step row is updated in place (`attempt + 1`). Do not append a second row for the same step name.
- Button only when run status is failed. Confirm uses the shared `data-confirm` modal (mutations may already have succeeded).
- After confirm: `HX-Redirect` to the existing live page.
- Unpublished definition, renamed/deleted failed step, or purged event: toast and refuse. Paused (`enabled: false`) still retries: `ExpandDefinitions` keeps the published def loaded with `Enabled: false`. Event/webhook/cron fire paths skip `!Enabled`; `ExecuteManual` refuses a paused def. Salvage is retry-only.
- `RetryFailedRun` does not call `IncResume` (that metric is for resume-from-checkpoint).
- `UpdateRunStatus` to start clears `completed_at` and error (also needed for `ResumePipeline`). Done also `SetError("")` so a successful retry does not keep the previous run error.
- Web UI only. No YAML automatic `retry` and no CLI.

`ClaimFailedRun` is a CAS (`status = failed`). Parent matching uses `pipeline.RunBelongsToParent`. Event envelope mapping uses `store.DataEventFromRow`.

## Alternatives considered

- **New run ID.** Event dedup would block a second consume of the same event; a synthetic event would fork history from Run 73.
- **In-run automatic retry (YAML `retry`).** Wrong layer: this is operator salvage after an external API failure, not backoff inside one attempt.
- **Replay the stored param snapshot.** Would ignore published YAML fixes (for example a corrected `column_id`).

## Consequences

Retrying a mutation that actually succeeded (JSON-RPC `false` after create, timeout after write) can duplicate side effects. The confirm copy says so. Cron/manual runs with no `data_events` row cannot retry until that envelope is persisted.

## Verification

`TestEngine_RetryFailedRun_skipsSuccessfulSteps`, `TestEngine_RetryFailedRun_skipsDoneStepsWhenEarlierStepInserted`, `TestEngine_RetryFailedRun_runsWhenPublishedDefinitionIsPaused`, `TestRetryStartIndex`, and `TestEngine_RetryFailedRun_Errors` in `pkg/pipeline/engine_store_test.go`. `TestPipelineStore_ClaimFailedRun` / `PrepareStepRetry` / `UpdateRunStatus_StartClearsCompletedAt` / `UpdateStepRun_ClearsErrorOnDone` in `internal/store/store_facade_extra_test.go`. `TestPipelineRunSteps_showsConfirmOnFailedRun` and `TestRetryPipelineRun_redirectsToLivePage` in `internal/modules/web/pipeline_webservice_test.go`. `TestPipelineStepRunsDetail` confirm attributes in `pkg/views/partials/pipeline_runs_test.go`. Owning BDD: `tests/specs/pipeline_page_spec_test.go` (requires Docker; not claimed without `test:specs`).
