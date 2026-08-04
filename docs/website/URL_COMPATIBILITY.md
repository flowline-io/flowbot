# URL Compatibility Matrix

## Stable Entry Points

The following entry pages are intentionally preserved as stable URLs:

- `index.html`
- `design.html`
- `api.html`
- `tutorials.html`
- `skills.html`
- `404.html`

## Navigation Strategy

- Use relative links between website pages (`index.html`, `design.html`, and so on).
- Use `docs/getting-started/` as the docs entry point from website pages.
- Keep GitHub links absolute to avoid repository path ambiguity.

## Fallback Behavior

- If a route cannot be resolved, `404.html` provides recovery links to `index.html` and `docs/getting-started/`.
- Reordering navigation labels must not remove or rename the stable entry files above.
