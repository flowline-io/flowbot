# Agent Note: Agent eval runs on PR paths, a weekly schedule, and manual dispatch

Status: implemented

## Problem

`.github/workflows/agent-eval.yml` ran the regression eval suite on every push and every pull request that touched the same paths list. Both triggers covered the same commits, so a branch with an open PR paid for the eval twice, and the paths list had to be written twice for every new eval-related path. The push trigger also made agent-path pushes wait on a suite that testing CI already covers with the same unit tests.

## Decision

The workflow has three triggers: `pull_request` restricted to the eval paths, `schedule` at `0 3 * * 1` (UTC Monday 03:00, HKT 11:00) on the default branch, and `workflow_dispatch`. There is no `push` trigger, so a push never starts an eval run; the PR check and the weekly run are the gate and the drift detector.

## Alternatives considered

- **Keep the paths-filtered `push` trigger alongside `pull_request`.** Rejected: both fire for the same commit on a PR branch, so the eval job runs twice per update and the paths list stays duplicated.
- **Keep `push` and drop `pull_request`.** Rejected: the PR check is the pre-merge signal; a push-only trigger reports after the merge.
- **Nightly schedule instead of weekly.** Rejected: regression drift is slow and the suite is deterministic, so a daily run adds runner minutes without catching anything a weekly run would miss.

## Consequences

- A push to the default branch no longer runs eval; a regression merged on Monday morning surfaces at the next scheduled run or on a manual dispatch.
- The eval paths list is written once, in the `pull_request` trigger.
- `workflow_dispatch` runs against the selected branch, so an eval regression can be reproduced or checked before merging.

## Verification

- `.github/workflows/agent-eval.yml` declares `pull_request` with the eval paths, `schedule: 0 3 * * 1`, and `workflow_dispatch`.
- `docs/agent/README.md` states those trigger conditions next to the regression suite row.
- `actionlint .github/workflows/agent-eval.yml` passes.
