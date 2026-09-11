# Agent Note: DeepSeek V4.1 Flash catalog support

Status: implemented

## Problem

DeepSeek released V4.1 Flash under the API id `deepseek-flash`, with native vision and thinking mode. The catalog still listed `deepseek-v4-flash` / `deepseek-v4-pro` as text-only models, and OpenAI-compatible thinking used a `deepseek-v4` name substring that would not match `deepseek-flash`.

## Decision

- Catalog authority is `deepseek-flash` only (`Name`: DeepSeek V4.1 Flash). There are no `deepseek-v4-flash` or `deepseek-v4-pro` catalog entries.
- `deepseek-flash` has `ModalityImageIn` plus chat/tools/JSON features; context `1_048_576`, max output `384_000`.
- DeepSeek thinking is driven by catalog features `CapThinking` and `CapReasoningEffort` (both set on `deepseek-flash`). `isDeepSeekV4ReasoningModel` is gone. MiMo and Anthropic still use their existing name heuristics until a later migration.
- Migration guidance lives in this note and `docs/agent/developer-guide.md`. `docs/reference/config.yaml` has no DeepSeek example block.

## Alternatives considered

- Keep `deepseek-v4-flash` as the catalog id or as an alias — rejected; freezes a temporary API route as our contract and fights foundation-over-shims.
- Name-substring update (`deepseek-flash` / `deepseek-v4*`) without catalog features — rejected; repeats the coupling that broke on rename.
- Migrate MiMo/Anthropic thinking to catalog in the same change — deferred to keep the Flash delivery scoped.

## Consequences

- YAML still using `deepseek-v4-*` is an unknown model: default 128k context, no vision gate from catalog, no thinking transport / `reasoning_effort`.
- `pkg/agent/llm` depends on `pkg/agent/model` for `HasFeature` when building thinking profiles.
- `CapThinking` currently also gates OpenAI-compatible `reasoning_content` round-trip for catalog models; MiMo remains name-based.

## Verification

```bash
go tool task lint
go test ./pkg/agent/model ./pkg/agent/llm ./pkg/agent/loop ./pkg/agent/eval ./pkg/config ./internal/server/chatagent ./internal/store ./pkg/views/partials -count=1
```
