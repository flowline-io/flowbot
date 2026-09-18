# Agent Note: Server image CI smoke

Status: implemented

## Problem

`docker.yml` pushed `ghcr.io/flowline-io/flowbot` on `v*` tags with no post-push check. A missing sibling CLI, `composer`, `dcg`, or HEALTHCHECK `wget` only showed up after someone pulled the release. Sibling image workflows already smoke after push.

## Decision

After push, [`.github/workflows/docker.yml`](../../../../.github/workflows/docker.yml) pulls the semver tag (`v1.2.3` → `:1.2.3`), prints `docker image inspect` Size, and runs a one-shot container that checks baked artifacts: `/opt/app/flowbot`, `/opt/app/flowbot-cli_linux_amd64`, `composer --version`, `dcg --version`, `/etc/dcg/config.toml`, and `wget` on PATH.

The smoke does not start the HTTP server. `/livez` needs PostgreSQL, Redis, and `flowbot.yaml`; that path stays with compose / deploy, not this job. Size is observational (no ceiling).

## Alternatives considered

- **Build, smoke locally with `load: true`, then push.** Rejected: sibling workflows smoke the published GHCR tag so the job fails on a registry/tag mismatch, not only a local load.
- **Start Postgres + Redis and curl `/livez`.** Rejected for this job: that is a stack smoke, not an image-contents smoke, and it needs a CI config plus services the Dockerfile does not bake.
- **Override ENTRYPOINT and run `/opt/app/flowbot` expecting a config error.** Rejected: fx startup is not a stable contract, and a missing binary is already caught by `test -x`.

## Consequences

- A red smoke leaves the GHCR tag in place (same as sandbox / Presidio / CDP). Operators must not treat a failed job as “the tag was never published.”
- Adding a baked binary or HEALTHCHECK tool to the server image means extending this smoke in the same change.

## Verification

`.github/workflows/docker.yml` has `Resolve smoke test tag` then `Smoke test image` after `Build and push server`. The run step pulls `ghcr.io/flowline-io/flowbot:${{ steps.smoketag.outputs.value }}` and asserts the artifact list above. Image contents remain [deployments/Dockerfile](../../../../deployments/Dockerfile).
