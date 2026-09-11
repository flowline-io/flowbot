# Security Policy

## Supported versions

Flowbot is pre-1.0. Security fixes target the latest release on the `master` branch and the most recent GitHub Release tag. Older tags are best-effort only.

| Version | Supported |
| --- | --- |
| Latest release (`v0.x`) | Yes |
| `master` tip | Yes |
| Older release tags | Best effort |

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Prefer a private report via GitHub Security Advisories:

1. Open [Report a vulnerability](https://github.com/flowline-io/flowbot/security/advisories/new).
2. If private vulnerability reporting is not yet enabled on the repository, contact an organization owner through [github.com/flowline-io](https://github.com/flowline-io) and ask them to enable **Private vulnerability reporting** under repository settings, then use the advisory form above.

Include:

- Affected version or commit
- Impact (confidentiality, integrity, availability)
- Reproduction steps or a minimal proof of concept
- Whether the issue is already public elsewhere

We aim to acknowledge reports within **7 days** and to share a remediation plan or status update within **30 days**. Timing may vary for complex issues.

Please give us a reasonable window to ship a fix before any public disclosure.

## Scope (examples)

In scope:

- Remote code execution, auth bypass, privilege escalation
- Injection into pipeline / workflow / agent surfaces that reaches host or sandbox escape
- XSS or unsafe HTML rendering of user- or agent-controlled content (see `pkg/utils.MarkdownToSafeHTML` in the developer guide)
- Credential or secret leakage in logs, APIs, or releases
- Dependency vulnerabilities with a credible exploit path in this project

Out of scope (unless they enable a real attack):

- Denial of service from unbounded self-hosted load without a distinct bug
- Issues that require already-compromised admin credentials on a single-admin homelab deploy
- Reports that only affect misconfigured third-party providers outside Flowbot's control

## Maintainer notes

Repository admins should keep **Private vulnerability reporting** enabled under Settings → Code security. After a fix ships, publish a GitHub Security Advisory and mention the issue under `### Security` in [CHANGELOG.md](CHANGELOG.md).
