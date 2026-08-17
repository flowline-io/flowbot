# Agent Note: Remap GitHub source links in Pages lychee

Status: implemented

## Problem

The Pages workflow runs lychee against the just-built `docs/website` HTML. Entry pages link to this repo with `https://github.com/flowline-io/flowbot/blob/master/...` (and one `tree/master` path). Lychee's GitHub token only covers repository-root URLs; blob and tree HTML is fetched as ordinary pages. GitHub returns 503 (and intermittent 404) under that load, so a deploy of existing files fails even when the paths exist in the checkout.

## Decision

The lychee step remaps `https://github.com/flowline-io/flowbot/(blob|tree)/master` to `file://${{ github.workspace }}`, the same local-file pattern already used for `https://flowline-io.github.io`. Missing source files still fail; GitHub HTML rate limits do not. Repository-root `https://github.com/flowline-io/flowbot` links stay remote (the token covers that shape). The workflow path filter includes `.github/workflows/pages.yml` so a remap-only change still runs the job.

## Alternatives considered

- **Pass `GITHUB_TOKEN` / `--github-token`.** Rejected as the fix: the action already defaults to `github.token`, and that client does not check blob or tree file paths.
- **`--accept 503` (or 404).** Rejected: it would hide real missing Pages or third-party URLs, and 404 mixed with 503 on the same blob URL is GitHub flakiness, not an acceptable success code.
- **`--exclude` github.com.** Rejected: a typo in a source-file href would no longer fail the job.

## Consequences

A broken GitHub blob/tree href to a path that is not in the commit still fails, but as a local file miss after remap. Links to other GitHub repos, or to this repo without `/blob/master` or `/tree/master`, are still fetched over HTTP.

## Verification

`.github/workflows/pages.yml` `Verify internal links` passes the github.io remap and the blob/tree remap to `lychee`. How lychee is installed is in [pages-lychee-binary-install](2026-08-17-pages-lychee-binary-install.md). The blob/tree targets linked from entry HTML exist in the workspace.
