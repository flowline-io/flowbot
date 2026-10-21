# Agent Note: Remove uncalled providers

Status: implemented

## Problem

AdGuard Home, ArchiveBox, n8n, Slash, and Dropbox each had a provider package and a `vendors` block in the reference config. Production code never imported AdGuard, ArchiveBox, or n8n. Slash was only reached from `module.Shortcut`, and nothing called `Shortcut`. Dropbox was registered for OAuth and had no later upload or token use. That is the same shape as the removed Pocket, Safeline, and Wallos stubs: a config surface without a call chain. See [remove instruct and ghost configs](2026-10-21-remove-instruct-and-ghost-configs.md).

## Decision

Flowbot does not ship clients for AdGuard Home, ArchiveBox, n8n, Slash, or Dropbox.

Removed with them: the five `pkg/providers` packages, their `vendors` blocks in the reference config, `module.Shortcut`, and the Dropbox `fx.Invoke` registration. OAuth production providers are GitHub and Slack.

The ArchiveBox name stays on the homelab probe fingerprint (`Provider: "archivebox"`, capability `archive`) and in the discovery guide. That label identifies a compose app; it does not load a client.

The config loader ignores leftover keys under `vendors`. Operators drop `vendors.adguard_home`, `vendors.archivebox`, `vendors.n8n`, `vendors.slash`, and `vendors.dropbox` when they want `flowbot.yaml` to match the template.

## Alternatives considered

- Keep the clients behind empty config — rejected. Pre-release prefers one config surface, and these packages have no product caller.
- Delete the ArchiveBox discovery fingerprint with the client — rejected. The fingerprint is a homelab label, not a call into the client. Compose apps can still be tagged `archive`.
- Leave Dropbox registered for a later upload path — rejected. Registration without a caller is the same unused surface as the HTTP clients.

## Consequences

- `pkg/providers` is 24 packages. GitHub and Slack are the OAuth providers wired in `internal/server/providers.go`.
- Homelab discovery still fingerprints ArchiveBox as capability `archive`.
- Existing `flowbot.yaml` files may still contain the five vendor blocks. The loader does not fail on them.

## Verification

- `go test ./pkg/module/ ./pkg/providers/... ./internal/server/ ./pkg/homelab/...`
- `go tool task lint`
