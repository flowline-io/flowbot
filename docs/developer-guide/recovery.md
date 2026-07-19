# Recovery Manager

How Flowbot recovers incomplete work after a restart.

## Pipeline incomplete runs

When a process stops mid-pipeline, run rows may remain in a non-terminal state (`running` / `pending`).

On a healthy restart:

1. Event history remains in PostgreSQL (`data_events` + `event_outbox`).
2. Redis Streams are rebuilt/consumed from durable outbox records — Redis is not the sole store.
3. Incomplete pipeline runs can be inspected in the Web UI under Pipelines → Runs, or via the API/CLI.

### Operator steps

1. Confirm `/readyz` is healthy (PostgreSQL + Redis).
2. Open the pipeline run that was interrupted; check which step was last successful.
3. Re-trigger the pipeline or reconcile the failed step manually if the run is stuck.
4. If Redis was wiped, restart Flowbot and allow the outbox publisher to republish pending events; do not truncate `data_events` unless you intend to drop history.

### What is not automatic

- Flowbot does not silently invent a replacement run for every interrupted execution.
- Idempotency keys prevent double application of the same event; re-emitting a new event creates a new run.

## Workflow runs

Workflow DAG tasks follow the same persistence model (run/step rows in PostgreSQL). Resume behavior depends on the workflow engine; prefer inspecting the run UI after an unexpected restart.

## Related

- [Self-hosting](../self-hosting.md) — backup/restore
- [Pipeline user guide](../user-guide/pipeline.md)
- [Monitoring](monitoring.md)
