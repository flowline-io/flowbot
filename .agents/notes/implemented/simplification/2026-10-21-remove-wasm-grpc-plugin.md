# Agent Note: Remove wasm and gRPC plugins

Status: implemented

## Problem

`pkg/plugin` was a second extension path beside in-tree modules, capabilities, and providers. Wasm (wazero) and gRPC (`hashicorp/go-plugin`) runners could register a module, an ability, or a provider from an out-of-process or in-process binary. Nothing in the product shipped a plugin, the system defaulted to disabled, and the host API duplicated registries that already have a Go extension point. Keeping it meant a config block, a startup manager, and two runtime dependencies for a path homelab installs do not use.

## Decision

Flowbot does not load wasm or gRPC plugins.

Removed with them: `pkg/plugin` (manifest, manager, local source, wasm runner, gRPC runner and SDK, module/ability/provider adapters), the `plugins` config section, and the settings copy for that section. In-tree modules, capabilities, providers, and notify providers stay.

The config loader ignores a leftover `plugins:` block. The wazero `NewRuntime` race workaround is archived at [.agents/notes/archived/bug-fix/2026-09-15-wazero-newruntime-version-race.md](../../archived/bug-fix/2026-09-15-wazero-newruntime-version-race.md).

## Alternatives considered

- Keep the manager behind `plugins.enabled: false` — rejected. A disabled runtime still has a host ABI, config, and dependencies, and pre-release prefers one extension path.
- Delete only the wasm runner and keep gRPC — rejected. Both runtimes implement the same unused host contract; one leftover runner would still own the manager and config.

## Consequences

- Extension is in-tree Go: modules, `pkg/capability`, `pkg/providers`, and `pkg/notify`.
- `wazero`, `hashicorp/go-plugin`, and the direct gRPC/protobuf module requirements leave with the plugin tree.
- Operators drop a `plugins:` block from `flowbot.yaml` when they want the file to match the schema. The loader does not fail on that block.

## Verification

- `go test ./internal/modules/... ./pkg/config/...`
- `go tool task lint`
- Config field docs and `settings_desc.zh.toml` are regenerated from the remaining config types.
