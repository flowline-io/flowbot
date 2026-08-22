# Agent Note: Web search DuckDuckGo keyless fallback

Status: implemented

## Problem

`web_search` required `chat_agent.web_search.api_key` (SerpApi). Homelab and local agent runs without that paid key could not search the web at all, even though the tool is on the default coding/scout allowlists.

## Decision

When `APIKey` is set, `WebSearchTool` still calls SerpApi Google Search. When it is empty, the tool queries the keyless DuckDuckGo HTML endpoint (`https://html.duckduckgo.com/html/`), unwraps `uddg` redirect URLs, and formats the same title/URL/snippet hits. `BaseURL` still overrides the selected backend for tests.

The HTML parser follows the DuckDuckGo markup used by [pigo](https://github.com/smallnest/pigo/blob/master/internal/agenttool/websearch_backends.go): `<a class="result__a">` for title and href, `result__snippet` by position.

## Alternatives considered

- **Keep SerpApi required.** Blocks the tool for any install without a third-party key.
- **Add Tavily/Brave credential backends as in pigo.** Extra yaml/env surface; Flowbot already has SerpApi as the paid path.
- **DuckDuckGo Instant Answer JSON.** Instant answers are not a substitute for organic web hits.
- **Scrape Google HTML.** Fragile and against Google's terms.

## Consequences

- DuckDuckGo HTML markup can change and yield empty results without a hard error.
- Result ranking and snippets differ from SerpApi Google; operators who need Google quality still set `api_key`.
- The fallback is an unauthenticated third-party request; bot detection or rate limits surface as HTTP/parse errors.

## Verification

- `pkg/agent/tools/coding/web_search_parse_test.go`: DDG HTML parse, `uddg` unwrap, missing snippet.
- `pkg/agent/tools/coding/tools_test.go`: empty `APIKey` hits the HTML backend; SerpApi path still used when the key is set.
- `docs/reference/config.yaml` and `ChatAgentWebSearchConfig` godoc treat `api_key` as optional.
