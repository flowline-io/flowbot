# Capability Webhooks

Inbound provider webhooks that convert third-party payloads into `DataEvent` records.

Source: `pkg/capability/` (`WebhookConverter`, `EventSourceManager`)

## Endpoint

```
POST /webhook/provider/*
```

The `*` segment is the value returned by each capability's `WebhookPath()`. Most providers use `{provider}/events`:

```
POST /webhook/provider/{provider}/events
```

| Capability | Full path |
| ---------- | --------- |
| github | `/webhook/provider/github/events` |
| gitea | `/webhook/provider/gitea/events` |
| miniflux | `/webhook/provider/miniflux/events` |
| karakeep | `/webhook/provider/karakeep/events` |
| memos | `/webhook/provider/memos/events` |
| kanboard | `/webhook/provider/kanboard/events` |
| example | `/webhook/provider/example` |

This route is separate from pipeline webhook triggers (`POST /webhook/{path}`).

## Authentication

There is no shared auth middleware. Each `WebhookConverter.VerifySignature` implements the provider's scheme. Failures return **401**. An empty `webhook_secret` / `webhook_token` in config rejects all deliveries. Query parameters are exposed to verifiers as `X-Query-*` headers.

Flowbot does not enforce a minimum or maximum length on secrets or tokens.

| Capability | Method | Credential location | Request side |
| ---------- | ------ | ------------------- | ------------ |
| GitHub | HMAC-SHA256 | `vendors.github.webhook_secret` | `X-Hub-Signature-256: sha256=<hex>` |
| Gitea | HMAC-SHA256 | `vendors.gitea.webhook_secret` | `X-Gitea-Signature: <hex>` |
| Miniflux | HMAC-SHA256 | `vendors.miniflux.webhook_secret` | `X-Miniflux-Signature: <hex>` |
| example | HMAC-SHA256 | `vendors.example.webhook_secret` | `X-Signature: <hex>` |
| Karakeep | Bearer token | `vendors.karakeep.webhook_token` | `Authorization: Bearer <token>` |
| Memos | Bearer token | `vendors.memos.webhook_token` | `Authorization: Bearer <token>` |
| Kanboard | Query token | `vendors.kanboard.webhook_token` | `?token=<token>` (read as `X-Query-Token`) |

HMAC providers sign the raw request body with the configured secret and compare against the signature header using constant-time equality. Bearer and query-token providers compare the configured value directly (body is not signed).

## Processing flow

1. Match path to a registered `WebhookConverter`.
2. Collect headers and query args; run `VerifySignature`.
3. On success, `Convert` the body into one or more `DataEvent` records.
4. Emit events asynchronously (persist via the event source emitter).

See `docs/reference/config.yaml` under `vendors.*` for the credential keys.
