# Agent Note: BDD CI image pull fallbacks

Status: implemented

## Problem

GitHub-hosted runners share egress IPs. Anonymous pulls from both Docker Hub and AWS Public ECR are rate-limited (`toomanyrequests`). Pinning the BDD job to `public.ecr.aws/docker/library/*` only moved the failure: Postgres often succeeded, Redis then hit ECR's anonymous quota, and AND-retries re-queried the same registry after a few seconds.

## Decision

The BDD job in `.github/workflows/testing.yml` pre-pulls each image independently. For each image it tries AWS Public ECR, Google's Docker Hub mirror (`mirror.gcr.io`), then Docker Hub, and tags the winner to `postgres:16-alpine` / `redis:7-alpine` (the spec defaults). Redis tries GCR first so the two pulls do not spend the same ECR anonymous quota back-to-back. Backoff runs only after every source fails. Specs keep reading `POSTGRES_IMAGE` / `REDIS_IMAGE`; local `test:specs` is unchanged.

## Alternatives considered

- **Authenticate to ECR Public with AWS credentials.** Rejected: secrets and IAM for a public OSS CI job.
- **Docker Hub login or OIDC.** Rejected: requires a Docker org and stored or federated credentials.
- **Cache `docker save` tarballs in `actions/cache`.** Rejected for now: extra moving parts, and a cache miss still needs a pull path.
- **Stay on ECR-only with longer sleeps.** Rejected: one registry; shared-IP quota can last minutes.
- **Mirror official images into GHCR.** Rejected: operational burden for two tags.

## Consequences

- CI no longer depends on a single anonymous registry quota.
- A successful pull is not retried (`docker image inspect` before pull).
- Local defaults stay Hub tags; only CI pre-pulls from mirrors.

## Verification

`.github/workflows/testing.yml` BDD job: independent `pull_one` with three sources, tag to spec defaults. `tests/specs/lifecycle.go` still defaults to `postgres:16-alpine` and `redis:7-alpine`. CI mechanics: [docs/testing/bdd-specs.md](../../../../docs/testing/bdd-specs.md).
