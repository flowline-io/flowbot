# Agent Note: DeepSource checks for first-party public JS/CSS

Status: implemented

## Problem

DeepSource only ran `go` and `shell`. First-party UI assets under `public/js` and hand-written `public/css` had no PR-side static analysis, while local lint already covers JS via oxlint and formatting via oxfmt. CSS had no equivalent DeepSource or local static gate.

## Decision

- Enable DeepSource `javascript` and `css` analyzers in [`.deepsource.toml`](../../../../.deepsource.toml).
- Treat DeepSource as a PR-side supplement; local authority for JS style/format stays `oxlint` / `oxfmt` (`go tool task lint:js` / `format`). Do not enable DeepSource code formatters.
- Scope first-party embedded UI only via top-level `exclude_patterns`: `public/vendor/**`, generated `public/css/app.css`, and `docs/website/**`.
- JavaScript meta: `environment = ["browser"]`, explicit vendor/app globals, no `style_guide` or framework `plugins`, `cyclomatic_complexity_threshold = "very-high"`, and `skip_doc_coverage` for common function forms.
- Keep JS/CSS findings advisory at first (do not require them as a merge check) until noise is reviewed.

## Alternatives considered

- **Replace oxlint with DeepSource as the JS authority.** Rejected: no frontend `package.json`, and dual style systems would conflict with the existing task loop.
- **Analyze all of `public/` including vendor and `app.css`.** Rejected: third-party and Tailwind/DaisyUI bundles produce noise without maintainer value.
- **Include `docs/website` in the same analyzers.** Rejected: marketing site conventions differ from the ops console; out of scope for this change.
- **Block merges on JS/CSS issues immediately.** Deferred: large first-party scripts would create an unrelated cleanup tax on unrelated PRs.
- **Enable DeepSource Prettier.** Rejected: `oxfmt ./public` already owns formatting.

## Consequences

- DeepSource reports bugs/security/anti-patterns on first-party `public/js` and hand-written CSS (`custom.css`, `styles.css`, `clip.css`, `chatagent-markdown.css`).
- Vendor scripts/styles, generated `app.css`, and `docs/website` assets are not analyzed.
- Merge gates for JS/CSS stay optional until maintainers tighten them in the DeepSource dashboard.

## Verification

```bash
# Config is declarative; DeepSource runs after the file reaches the default branch.
# Confirm analyzers and exclude_patterns in the DeepSource dashboard under Settings > Code Review.
test -f .deepsource.toml
```
