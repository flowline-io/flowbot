# Agent Note: MCP Server Card

Status: implemented

## Problem

A client that already has the Streamable HTTP URL still has no standard document for the server's name, version, and protocol before it opens a session. SEP-2127 is Final and reserves `GET <streamable-http-url>/server-card` for that document. Waiting for a client to require it leaves the reserved path empty, and putting a tool list on a public document would publish a scope-shaped catalog that the live `tools/list` call is supposed to filter.

## Decision

`GET /mcp/server-card` is mounted with `/mcp` when `mcp.enabled` is true, unless `mcp.server_card` is false. The route does not authenticate. The body is the Server Card from `pkg/mcp` and lists no tools. The remote origin is `flowbot.url` when that value is set; otherwise it is the request host, with `X-Forwarded-Proto: https` or a TLS request selecting https. The field list and the pre-release schema caveat live in [mcp.md](../../../docs/user-guide/mcp.md).

`flowbot mcp config` prints the card URL for the endpoint it was given, then the snippet, both on stdout. `-o json` puts the URL on `data.server_card`. That command does not read `mcp.server_card`; it describes the URL the operator passed. `/service/web/settings/mcp` shows the URL only while this process publishes the card. The extension repository is still Experimental. No mainstream client was found that already reads a Server Card, so this stays off the v1.0 critical path.

## Alternatives considered

- Hosting the card at `/.well-known/mcp-server-card`. Rejected: SEP-2127 reserves the suffix on the Streamable HTTP URL. `.well-known` is for the AI Catalog, which this change does not publish.
- Including `tools` or an `Authorization` header template. Rejected: the SEP omits primitives because they depend on the caller, and a public tool list would expose the catalog that token scopes filter on `tools/list`. Header templates are connection prompts, not identity, and the client snippets already carry the bearer header.
- Holding the path until a client reads it. Rejected: the name, version, endpoint, and protocol version already exist, and the reserved path is the cheap place to publish them. The separate `mcp.server_card` switch turns the public document off without disabling `/mcp`.
- Caching headers and `Access-Control-Allow-Origin: *` on the card response. Rejected for this cut: the request was a static JSON document, and those headers are not required to publish the path.

## Consequences

The card is advisory. Clients that never fetch it keep using the configured `/mcp` URL and a header token. Operators who do not want a public JSON document set `mcp.server_card: false` and restart; the settings page then omits the URL. A schema break in the extension is a card-document change, not an `/mcp` protocol change. `flowbot mcp config` still prints `<endpoint>/server-card` for the URL it was given, including when that server has the card turned off.

## Verification

`pkg/mcp` tests the document and the card URL. `internal/server/mcp` tests the unauthenticated GET, `flowbot.url`, forwarded https, a JSON problem response, and both off switches. `cmd/cli/command` tests the stdout line ahead of the snippet. `internal/modules/web` tests that the settings page shows the URL only while the card is published. BDD coverage is `tests/specs/mcp_spec_test.go` and `tests/specs/mcp_settings_spec_test.go`. Operator text: [mcp.md](../../../docs/user-guide/mcp.md).
