# Agent Note: Homelab discovery host-side contracts

Status: implemented

## Problem

Homelab discovery documented three "limitations" that mixed unfinished surface with
product intent: container-network probe resolution (config reserved, unimplemented),
fingerprint fields beyond path (type surface and docs implied more than the matcher
contract), and multi-capability compose bindings. Leaving half-finished knobs and
"future" matcher fields taught the wrong deployment model (Flowbot on the host,
labels as the contract).

## Decision

- Probes target **host-published TCP ports only** (`Host` or `localhost` + host port).
  Container-network resolution is permanently out of scope.
- Remove `probe_networks` and `probe_port_strategy` from config and discovery types.
- Fingerprints are **path-reachability only** (`ServiceFingerprint.Paths`); header,
  title, and body_key matching are not part of the contract.
- One compose file binds **one** capability (existing `ParseLabels` behavior elevated
  to a documented contract).
- `flowbot.endpoint.base` must be reachable from the Flowbot process; docs use
  `http://127.0.0.1:<published-port>/...`.
- Runtime probes remain optional best-effort; labels stay authoritative when
  `label_priority` is true.
- User-guide capability examples use canonical provider IDs (`karakeep`, `miniflux`,
  `kanboard`, …). Legacy domain labels still parse with a deprecation warning;
  `flowbot.backend` remains ignored.
- Auth header names and unpublished ports are probe limits documented as contracts
  in the user guide Design Decisions table (no separate Limitations section).

## Alternatives considered

- Keep `probe_networks` / `container`/`both` strategies as reserved — rejected;
  unfinished config surface is worse than a smaller honest API under pre-release
  foundation-over-shims.
- Implement Docker-network / container-IP probing while Flowbot stays on the host —
  rejected; expands security and ops surface for a secondary discovery path.
- Keep or finish header/title/body fingerprint matching — rejected; path matching
  is enough for known services, and a narrower contract avoids silent half-matches.
- Multi-capability labels or per-service scan granularity — rejected until a real
  multi-capability compose scenario forces it; one-compose-one-capability stays clear.
- Leave user-guide examples on legacy domain labels — rejected; examples should match
  the canonical `knownCapabilities` keys in `pkg/homelab/labels.go`.

## Consequences

- User configs that set `probe_networks` or `probe_port_strategy` are ignored by
  mapstructure (fields removed).
- Contract home for operators: [homelab-discovery.md](../../../docs/user-guide/homelab-discovery.md)
  (links here for rationale). Config sample: [config.yaml](../../../docs/reference/config.yaml).
- Code: `pkg/homelab/discovery.go`, `pkg/homelab/probe/{engine,fingerprints,http_probe}.go`,
  `pkg/config/config.go`, `internal/server/homelab.go`.

## Verification

`go test ./pkg/homelab/probe/ ./pkg/homelab/ ./internal/server/` and `go tool task lint`
pass with host-published targeting, path-only fingerprints, and the removed discovery
config fields absent from `HomelabDiscovery` / `DiscoveryConfig`.
