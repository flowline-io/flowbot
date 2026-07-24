# Platforms Guide

Chat integrations implementing `protocol.Driver`, `Adapter`, and `Action`. Reference: `internal/platforms/slack/`.

## Entry points

- Core: `caller.go`, `convert.go`, `registry.go` (`PlatformRegister`, `GetCaller`, `MessageConvert`)
- Wired today: `slack/` (`fx.Provide`), `tailchat/` (controller construction + router)
- Not wired: `discord/` (package + config exist; no server fx / platform callback)

Per platform: `driver.go`, `adapter.go`, `action.go`, `types.go` (`const ID`).

## Boundaries

- Never import `pkg/providers/*`
- Platform-native conversion stays in the adapter; routing in `action.go`, not `server/`
- Errors: `protocol.NewFailedResponse` / `NewSuccessResponse`
- Wire each enabled platform in `internal/server/fx.go` (or controller) and `router.go` as needed
- Lifecycle: `handlePlatform` starts `WebSocketClient` on fx Lifecycle

## Testing

Table-driven unit tests; mock SDKs or `httptest`.
