# Agent Note: Install lychee as a release binary in Pages CI

Status: implemented

## Problem

The Pages job failed before any step ran: the runner could not download `lycheeverse/lychee-action@v2` from `codeload.github.com` (HTTP 429 after three retries). First-party actions (`actions/checkout`, `actions/setup-go`) had already been fetched. That download is outside the workflow's control; a link-check remap cannot run if the action never installs.

## Decision

Pages installs a pinned lychee release (`lychee-v0.24.2`) with authenticated `gh release download` and retries, then runs `lychee` in a `run` step. It does not `uses:` `lycheeverse/lychee-action`. Link remaps stay as in [pages-lychee-github-source-remap](2026-08-17-pages-lychee-github-source-remap.md). `GITHUB_TOKEN` is passed to lychee for repository-root GitHub URLs.

## Alternatives considered

- **Re-run until GitHub recovers.** Rejected as the only fix: every Pages run still has to fetch that third-party action at job setup, with no retry we own.
- **Vendor `lychee-action` in-repo.** Rejected: the job needs the lychee binary, not the action wrapper.
- **`cargo install lychee`.** Rejected: requires a Rust toolchain and is slower than a release tarball.
- **Skip or `continue-on-error` the link check.** Rejected: it drops the gate.

## Consequences

Lychee version is pinned in `.github/workflows/pages.yml` (`LYCHEE_TAG`); bumps are a workflow edit. `peaceiris/actions-gh-pages@v4` remains a third-party action download for the publish step.

## Verification

`.github/workflows/pages.yml` has an `Install lychee` step using `gh release download` and a `Verify internal links` step that invokes `lychee` directly. The file does not reference `lycheeverse/lychee-action`.
