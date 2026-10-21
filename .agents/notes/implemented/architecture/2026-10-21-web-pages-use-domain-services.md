# Agent Note: Web pipeline, workflow, and function pages use domain services

Status: implemented

## Problem

The pipeline, workflow, and function pages in `internal/modules/web` wrote and read definitions, runs, and stats by calling `internal/store` directly. `pkg/pipeline`, `pkg/workflow`, and `pkg/functions` already owned apply, publish, delete, and run for the JSON APIs. The pages duplicated that path, so a reload, a version check, or a draft rule could change in one place and not the other.

## Decision

`pipeline_webservice.go`, `workflow_webservice.go`, and `function_webservice.go` call the process-wide `ActiveService` for those three domains. Missing page reads and writes are methods on those services. Persistence stays behind the existing catalog and run-store interfaces. Store adapters map ent rows to `pkg/types/model` and the existing stats types.

Pipeline runs and steps use `pipeline.RunStore`, attached with `SetRunStore`. Retry loads the original data event through `pipeline.EventLoader`, attached with `SetEventLoader`. `NewService` keeps its catalog-only constructor. Workflow step reads are on `workflow.WorkflowRunStore`. Workflow triggers keep their database ids on the catalog, because `WorkflowMetadata` triggers do not. Function stats and published version numbers are on `functions.Catalog`.

Handlers still compose list rows, map HTTP and HTMX, run the editor step test through `capability.Invoke`, and stream live run progress from Redis. Hub capability lists and agent tool options stay in the handler.

Write methods reload inside the service. Rename, publish, and workflow enable or trigger toggles log a reload failure and still succeed. Pipeline enable returns the reload error. Delete uses the existing `Service.Delete`, which reloads.

Home, analytics, and view prefetch still call the store.

## Alternatives considered

- Call only the methods the services already had — rejected. Apply publishes immediately, and the editor saves a draft and publishes on a separate action.
- One query method per page that returns a view row — rejected. The services stay reusable by the JSON APIs, and the pages keep their own partials.
- Leave retry's event lookup in the handler — rejected. That kept an `EventStore` call in the pipeline page.

## Consequences

- Deleting a pipeline from the page reloads the engine. Previously the page delete did not.
- Tests and `MountForE2E` call `InstallPageServices` so the pages see `ActiveService`. Production wiring is `internal/server`.
- The BDD pipeline CRUD helper calls the same pipeline service.

## Verification

- `go test ./pkg/pipeline/ ./pkg/workflow/ ./pkg/functions/ ./internal/store/ ./internal/modules/web/ ./internal/modules/automate/ ./internal/server/`
- `go tool task lint`
