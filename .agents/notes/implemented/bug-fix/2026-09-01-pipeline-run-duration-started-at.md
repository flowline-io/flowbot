# Agent Note: Pipeline run duration uses started_at

Status: implemented

## Problem

The pipeline runs table showed multi-hour or multi-day durations for runs whose steps finished in milliseconds. Operators retried or resumed an old run ID; `created_at` stayed at the original create time while `completed_at` was stamped on the later attempt, so `completed_at − created_at` counted idle time between attempts. Failed steps also omitted `completed_at`, so step duration showed `-` even after a terminal failure.

## Decision

- UI duration and the Started column use `started_at` (fall back to `created_at` only when `started_at` is zero), matching workflow runs.
- `UpdateRunStatus` to `start` refreshes `started_at` (covers `ResumePipeline` and keeps retry reclaim consistent with [manual retry](../feature/2026-08-28-pipeline-failed-run-retry.md)).
- `UpdateStepRun` stamps `completed_at` for Failed as well as Done and Cancel.

## Alternatives considered

- **Keep `created_at` and show a separate “age” column.** Rejected: the column is labeled duration/started; operators need attempt wall time, not lifetime since first create.
- **Backfill historical failed steps without `completed_at`.** Rejected: display-only gap for old rows; new failures are correct without a one-shot migration.

## Consequences

Retried or resumed runs show attempt duration after the claim/resume. Pre-fix failed step rows without `completed_at` still render `-` until re-run. Stats buckets already used `started_at` and stay aligned with the table.

## Verification

`TestRunsDuration` / `TestStepRunsDuration` in `pkg/views/partials/pipeline_runs_test.go`. `TestPipelineStore_UpdateRunStatus_StartClearsCompletedAt`, `TestPipelineStore_ClaimFailedRun`, and `TestPipelineStore_PrepareStepRetry` in `internal/store/store_facade_extra_test.go` assert `started_at` refresh and failed-step `completed_at`.
