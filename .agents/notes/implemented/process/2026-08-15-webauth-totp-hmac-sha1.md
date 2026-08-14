# Agent Note: TOTP keeps HMAC-SHA1

Status: implemented

## Problem

GSC-G505 / gosec G505 blocklists `crypto/sha1`. `pkg/webauth` uses SHA-1 only inside HMAC for HOTP truncation (`hotp`), which is the default algorithm in RFC 6238 TOTP and in `otpauth://` URIs consumed by common authenticator apps. Replacing it with SHA-256/SHA-512 without a coordinated enrollment change would desync codes from those apps (many still assume SHA-1 even when the URI advertises another algorithm).

## Decision

Web UI TOTP continues to use HMAC-SHA1. The `crypto/sha1` import carries `#nosec G505` with that rationale. Provisioning URIs keep `algorithm=SHA1`. Password hashing, backup-code peppering, and key derivation stay on bcrypt / HMAC-SHA256 / SHA-256 — SHA-1 is not used as a general-purpose hash.

## Alternatives considered

- **Switch TOTP to SHA-256 or SHA-512.** Rejected for now: breaks interoperability with the default authenticator-app profile and forces every enrolled account to re-scan without a clear security win for OTP (HMAC-SHA1 remains acceptable for this construction).
- **Globally exclude G505 in `task gosec`.** Rejected: would hide unrelated SHA-1 imports elsewhere.

## Consequences

- G505 remains a real signal for any new `crypto/sha1` use outside this documented TOTP path.
- A future move to SHA-256 TOTP needs a migration (re-enroll or dual-verify) and an updated Agent Note — do not silently change `hotp` or the URI algorithm.

## Verification

`go tool gosec` on `./pkg/webauth/` reports Nosec for G505 on the annotated import. `go test ./pkg/webauth/` covers code generation and verify window behavior.
