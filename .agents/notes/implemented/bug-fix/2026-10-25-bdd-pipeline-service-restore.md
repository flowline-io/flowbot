# Agent Note: BDD specs restore the process-wide pipeline service

Status: implemented

## Problem

`tests/specs` shares one `pipeline.ActiveService` per Ginkgo process. The blueprint page spec replaced that service, then set it to nil in `AfterEach`. Pipeline CRUD routes call `ActiveService()` when the request arrives. After a blueprint spec in the same process, create, read, update, publish, and delete all returned HTTP 500, including the cases that should be 404 or 409. Ginkgo randomizes spec order across `--procs=4`, so the failure showed up only on some seeds.

## Decision

The blueprint page spec saves `ActiveService` and `ActiveBlueprintService` in `BeforeEach` and writes those values back in `AfterEach`. Pipeline CRUD calls `ensurePipelineService` before each example. That helper installs a service from `store.Database` only when the process-wide service is nil, and it does not stop or replace the workflow service.

## Alternatives considered

- **Call `InstallPageServices` again from every CRUD example.** That also stops and replaces the workflow service, so a CRUD example could drop workflow state another spec in the process still needs.
- **Give each spec its own pipeline service and stop using the process-wide value.** The HTTP handlers and the BDD routes both read `ActiveService()` on purpose, matching production. A second wiring path would drift from that.

## Consequences

A blueprint example no longer clears the service for later specs in the same process. CRUD still passes if some other spec has cleared it. Specs that need a nil service must set that up themselves and restore the previous value.

## Verification

`go test -tags integration -c` compiles `tests/specs`. The Docker acceptance suite is the runtime check; this workspace had no Docker daemon, so `go tool task test:specs:ci` was not re-run here.
