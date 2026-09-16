# Browser tools (CDP)

Chat agent interactive browsing via Chrome DevTools Protocol. Implementation lives in [`pkg/agent/browser`](../../pkg/agent/browser/) and [`pkg/agent/tools/browser`](../../pkg/agent/tools/browser/). Decision record: [.agents/notes/implemented/architecture/2026-09-16-chat-agent-browser-cdp.md](../../.agents/notes/implemented/architecture/2026-09-16-chat-agent-browser-cdp.md).

## Enable

```yaml
chat_agent:
  browser:
    enabled: true
    driver: lightpanda          # or playwright (Chromium CDP)
    endpoint: ws://lightpanda:9222
    allow_private: false
    allow_hosts: []             # host or CIDR exceptions
    timeout: 30s
```

Compose already defines a `lightpanda` service (image `ghcr.io/flowline-io/flowbot-lightpanda`). From the host for local debug, CDP is published on `127.0.0.1:9222`. Optional Chromium backend (`ghcr.io/flowline-io/flowbot-playwright-cdp`):

```bash
docker compose -f deployments/docker-compose.yaml --profile playwright-cdp up -d playwright-cdp
```

Then set `driver: playwright` and `endpoint: ws://playwright-cdp:9222` (or `ws://127.0.0.1:9223` from the host).

## Images / CI

| Image | Dockerfile | GHCR |
| ----- | ---------- | ---- |
| Lightpanda CDP | [`deployments/lightpanda/Dockerfile`](../../deployments/lightpanda/Dockerfile) | `ghcr.io/flowline-io/flowbot-lightpanda` |
| Chromium CDP | [`deployments/playwright-cdp/Dockerfile`](../../deployments/playwright-cdp/Dockerfile) | `ghcr.io/flowline-io/flowbot-playwright-cdp` |

Chromium binds DevTools to loopback inside the container; the playwright-cdp image proxies `0.0.0.0:9222` → `127.0.0.1:9223` with `socat` so Compose siblings and host port maps can reach CDP.

Release tags use **`cdp-v*`** (independent of server `v*` and sandbox `sandbox-v*`). Workflow: [`.github/workflows/docker-browser-cdp.yml`](../../.github/workflows/docker-browser-cdp.yml) (also supports manual `workflow_dispatch`).

```bash
git tag cdp-v1.0.0 && git push origin cdp-v1.0.0
```

## Tools

| Tool | Role |
| ---- | ---- |
| `browser_navigate` | Open an http(s) URL (SSRF gate) |
| `browser_snapshot` | Compact interactive tree with opaque `ref`s |
| `browser_click` / `browser_type` | Act by `ref` from the latest snapshot |
| `browser_scroll` / `browser_wait` | Page settle helpers |
| `browser_screenshot` | PNG for vision-capable models |

Workflow: navigate → snapshot → click/type by `ref` → snapshot again. Snapshot prefers the accessibility tree and falls back to a DOM-role tree. Refs are invalidated after every navigate.


Permission key: `browser` (default ask). Scheduled/autonomous overlays deny it.

## Session model

One isolated CDP browser context per chat agent run. Context is attached in `Service.Run` and closed when the run finishes. Parallel `browser_*` calls in the same run are serialized.

## Screenshots

`browser_screenshot` returns text metadata plus an image `MediaPart`. The LLM convert path keeps the tool result text and injects a follow-up human message with the image bytes so providers that reject images on the tool role still see the screenshot.

## Related

- [Agent Sandbox](./agent-sandbox.md) — coding image only (no Playwright variant)
- [Developer Guide](./developer-guide.md) — general tool authoring
