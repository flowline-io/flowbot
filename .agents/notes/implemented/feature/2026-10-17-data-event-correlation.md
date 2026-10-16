# Agent Note: Data event correlation

Status: implemented

## Problem

`data_events.correlation_id` and `causation_id` existed, but capability, poll, webhook, and pipeline writes left them empty. A later trace cannot group one MCP write with the events it caused.

## Decision

`AppendDataEvent` and `persistAndPublishDataEvent` call `types.EnsureCorrelation`. An event that already has a correlation keeps it. Otherwise the id is copied from context lineage, and an empty causation becomes the context event id. If the context has no correlation, the event gets a new `cor_<ULID>` and an empty causation.

A pipeline run, including resume, writes a missing correlation onto the trigger event and puts that id plus the trigger event id on the step context. Downstream events inherit both. MCP mutation `tools/call` mints one correlation before execute and stores it on the audit row. Watermill's bus correlation id is unchanged.

## Alternatives considered

Minting only inside pipeline steps would leave poll and webhook events ungrouped. Overwriting an event that already has a correlation would break a caller that set the pair on purpose. Using the Watermill UUID as the domain id would mix a transport id with the stored lineage id.

## Consequences

Scope-denied MCP calls are rejected before execute, so those audit rows still have an empty correlation. Resource-link edges, trace, and the flow aggregate are not in this change.

## Verification

`go test ./pkg/types/ -run TestEnsureCorrelation ./internal/store/ -run TestAppendDataEvent_FillsCorrelation`
