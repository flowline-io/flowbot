# Agent Note: upgrading.md and README MCP discoverability

Status: implemented

## Problem

Operators hitting pre-1.0 removals (gateway/agent, wasm/gRPC plugins, Life, instruct, automate delete scopes, `chat_agent.enabled`) had only `CHANGELOG.md` and scattered Agent Notes. `docs/upgrading.md` 404'd. The product README had zero MCP mentions while `docs/user-guide/mcp.md` was complete, so first-contact readers never saw `/mcp`, `flowbot mcp config`, or the Server Card.

## Decision

- Add [docs/upgrading.md](../../../docs/upgrading.md) as the operator checklist for tip-of-tree and 0.99.18 breakings, linking changelog, Agent Notes, mcp.md, analytics, and schema DROP docs. It does not duplicate those homes.
- Put one MCP sentence on the README hero intro: Streamable HTTP `/mcp`, `flowbot mcp config`, Server Card, link to mcp.md. List Upgrading (and MCP under User Guide) in the Documentation section.
- Point self-hosting Upgrade step 1 and the docs index at upgrading.md. Append an upgrading link at the bottom of `CHANGELOG.md` and as the GoReleaser GitHub release footer.

## Alternatives considered

- Keep upgrade steps only in CHANGELOG Breaking bullets — rejected. Operators need a stable checklist URL that outlives a single release tag body.
- Add a full MCP section to the README — rejected. mcp.md already owns the contract; the README only needs discoverability.
- Release footer only, no CHANGELOG bottom link — rejected. Readers of the checked-in changelog still need the checklist without opening a GitHub Release.

## Consequences

- New tip-of-tree breakings that affect operators update upgrading.md in the same change as the changelog entry.
- GitHub Releases from GoReleaser append the upgrading URL after the generated notes.

## Verification

- `docs/upgrading.md` exists and covers agent/gateway, automate delete scopes, wasm/gRPC, Life, instruct/ghosts, uncalled providers, and `chat_agent.enabled`.
- Root `README.md` matches `MCP` (and links `docs/user-guide/mcp.md`).
- `CHANGELOG.md` ends with the upgrading link; `.goreleaser.yaml` `release.footer` points at the same path on `master`.
