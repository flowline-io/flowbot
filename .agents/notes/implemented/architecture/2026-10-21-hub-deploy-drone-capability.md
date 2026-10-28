# Agent Note: Hub deploy is the drone capability

Status: implemented

## Problem

The hub `deploy` chat command created a Drone build by calling the Gitea and Drone provider clients from `internal/modules/hub`. Modules are interaction entry points, and provider calls belong in a capability adapter. Pipelines and MCP could not trigger the same deploy.

## Decision

`pkg/capability/drone` owns the deploy sequence: the signed-in Gitea username is the Drone namespace, and the repository name stays `deploy`. Notification stays the default template with a soft skip ([note](../simplification/2026-08-23-deploy-default-notify-template.md)). The hub command is `capability.Invoke` of `drone.deploy` with the caller uid. The capability registers when `providers.drone.endpoint` is set. Scopes are `service:drone:read` and `service:drone:write`.

MCP advertises `drone.deploy` as a write with `idempotentHint=false` and `openWorldHint=true`, and `drone.health` as a read. MCP does not advertise a `uid` argument; when a capability declares one, the server fills it with the token subject before invocation. A caller cannot notify another user. Pipeline and hub callers set `uid` themselves; those call paths are not MCP.

## Alternatives considered

- Leave the sequence in the hub command and only wrap `CreateBuild` — rejected. The module would still own namespace lookup and notification, so pipelines would not share one deploy.
- Add the operation to the devops aggregator — rejected. Devops covers monitoring backends. Drone is its own provider, so CapType is `drone`.
- A generic `create_build` that takes namespace and repository — rejected. The chat command is the single word `deploy`, and the namespace is always the Gitea user.

## Consequences

- An empty Drone endpoint leaves the capability unregistered. The chat command then fails with capability not found.
- An MCP caller always notifies the token subject; MCP does not advertise `uid`, so a call cannot redirect the deploy notification.
- A missing default notify template or channel is still a soft skip after the build exists. Other notify errors still fail the command after the build is created.
- Workflow skill docs list `capability:drone.deploy` and `capability:drone.health`. There is no Drone CLI.

## Verification

`go test ./pkg/capability/drone/ ./internal/modules/hub/ ./internal/server/ ./internal/server/mcp/ -count=1` and `go tool task lint`.
