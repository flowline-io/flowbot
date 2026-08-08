# Flowbot Website North Star

## Scope

- Target: `docs/website` static site and shared assets.
- Audience priority: homelab operators first, developers second, contributors third.
- Language policy: English-first single language across this website.

## Brand Message

Flowbot is the orchestration engine for homelab systems: discover apps, unify capabilities, and run reliable automation across services.

## Narrative Model

Use a linear story for first contact pages:

1. Problem (fragmented apps and interfaces)
2. Solution (unified capability and workflow model)
3. Proof (architecture, API, tutorials, and skills)

## Information Architecture

- Home: value proposition and core product story.
- Product: design and API references.
- Learn: tutorials, skills, and docs entry.
- Community: GitHub project.

## Canonical Terminology

- Use these verbs for CTA and headings: Discover, Compose, Operate.
- Prefer "capability" over "integration endpoint".
- Prefer "workflow" for DAG execution and "pipeline" for event-driven automation.
- Prefer "control plane" and "runtime plane" wording where needed.

## Source of Truth Mapping

- `design.html`: consistent with `docs/architecture/`.
- `api.html`: consistent with `docs/api/swagger.yaml` and `docs/api/swagger.json`.
- tutorials and skills pages: only link to existing docs and generated content.

## SEO Goals

- Primary: technical SEO baseline (crawlable stable entry URLs, unique canonicals, robots + sitemap, basic Open Graph / Twitter cards).
- Secondary: intent-aligned discoverability for Design, API, Tutorials, Skills, and generated docs pages (per-page title/description/canonical; docs are not bulk-listed in the sitemap yet).
- Canonical host for this phase: `https://flowline-io.github.io` (see `URL_COMPATIBILITY.md`).
- Entry SEO is configured in `cmd/composer/action/webdoc/seo.yaml` and applied by `webdoc`.
- Do not expand information architecture for SEO; keep English-first single language.
- Search Console verification is out of scope for the site build; operators may add it later.

## Acceptance Thresholds

- Any core destination (Design, API, Docs, Tutorials, Skills) is reachable from Home within 2 clicks.
- Broken internal links: 0.
- Global navigation and footer taxonomy stay consistent on all website pages.
- Mobile navigation is usable and core content has no horizontal overflow.
- Stable entry pages expose canonical, description, and basic Open Graph / Twitter tags (via `webdoc` SEO injection).
- `robots.txt` allows crawling and declares the sitemap; `404.html` is `noindex`.
- Homepage includes a minimal `WebSite` JSON-LD block.
- Sitemap lists only the stable entry whitelist until an explicit expansion decision.
- Entry SEO copy and absolute URLs are owned by `cmd/composer/action/webdoc/seo.yaml`, not duplicated by hand across HTML files.
