# Agent Note: Chat agent browser tools over CDP

Status: implemented

## Problem

Chat agents needed interactive browsing (JS-rendered pages, click/type) beyond `web_fetch`. The agent-sandbox Playwright image variant bundled Chromium for shell/E2E inside ephemeral containers, which duplicated browser capability, inflated sandbox pulls, and never exposed a first-class tool contract to the Observe-Think-Act loop.

## Decision

Ship CDP-backed `browser_*` tools in the agent engine:

- Adapter and per-run session: [`pkg/agent/browser`](../../../../pkg/agent/browser/) using `github.com/chromedp/chromedp` against a remote CDP WebSocket (`NewRemoteAllocator` + isolated browser context). Driver values `lightpanda` and `playwright` share chromedp; `playwright` means Chromium remote-debugging CDP (compose profile), not Playwright Test.
- Tools: [`pkg/agent/tools/browser`](../../../../pkg/agent/tools/browser/); registered from chatagent only when `chat_agent.browser.enabled`. Not in `RegisterHeadless`.
- Session lifecycle: created in [`Service.Run`](../../../../internal/server/chatagent/service.go), attached via context, closed on defer; operations serialized with a per-session mutex; refs from `browser_snapshot` invalidate on navigate. Snapshot prefers CDP Accessibility tree (refs stamped on backend DOM nodes) and falls back to a DOM-role tree.
- Screenshots: tool results may include `MediaPart` images; [`transform.DefaultConvertToLLM`](../../../../pkg/agent/transform/convert.go) emits tool text plus a follow-up human message with binary image content. Dual-model routing still strips unsupported modalities via `SupportsModality`.
- Deploy: Compose `lightpanda` / optional `playwright-cdp` ([Dockerfiles](../../../../deployments/lightpanda/Dockerfile)); GHCR images `ghcr.io/flowline-io/flowbot-lightpanda` and `ghcr.io/flowline-io/flowbot-playwright-cdp` via [`.github/workflows/docker-browser-cdp.yml`](../../../../.github/workflows/docker-browser-cdp.yml) on `cdp-v*` tags. Playwright-cdp proxies container `0.0.0.0:9222` to Chromium loopback CDP with `socat`. Agent-sandbox Playwright stage and `playwright-*` GHCR tags are removed.


Navigate-time URL gating uses `allow_hosts` / `allow_private`. `allow_private` permits RFC1918 but still blocks loopback, link-local, and cloud-metadata unless allowlisted. A blocked final URL after redirect resets the page to `about:blank`. Page subresource fetches to private IPs remain a residual risk (no egress proxy in this change).

Permission key `browser` defaults to ask; scheduled overlay denies it.

## Alternatives considered

- Capability/provider wrap for CDP — rejected for v1; browsing is an agent runtime concern without hub/DB, same tier as `web_fetch`.
- Keep Playwright inside agent-sandbox and script via `run_terminal` — rejected; no stable tool contract, heavy image, and poor isolation vs a shared CDP service with per-run contexts.
- Raw CDP tool for the model — rejected; too large a surface and leaks Lightpanda vs Chromium differences.
- Hand-rolled WebSocket CDP client — rejected; chromedp is maintained and documented for Lightpanda.

## Consequences

- Operators who pulled `ghcr.io/flowline-io/flowbot-agent-sandbox:playwright-*` must switch to [Browser tools](../../../../docs/agent/browser.md) + Compose CDP.
- Browser tools appear only when enabled; missing CDP endpoint fails at first tool call, not at registry build.
- Non-vision tool models drop screenshot images after tool turns (text metadata remains).

## Verification

- `go test ./pkg/agent/browser/... ./pkg/agent/tools/browser/... ./pkg/agent/transform/... ./pkg/agent/permission/...`
- `go test ./internal/server/chatagent/ -count=1` (registry paths)
- Config sample: [`docs/reference/config.yaml`](../../../../docs/reference/config.yaml) `chat_agent.browser`
- Compose: `lightpanda` in [`deployments/docker-compose.yaml`](../../../../deployments/docker-compose.yaml); sandbox workflow builds `base` only; browser CDP images via `docker-browser-cdp.yml` (`cdp-v*`)
