# Agent Note: Enroll 2FA HTMX responses must be fragments

Status: implemented

## Problem

`POST /service/web/setup/2fa` is an HTMX swap (`hx-target="this"` / `hx-swap="outerHTML"` on the enroll form). Error and success handlers returned `Enroll2FAPage` / `BackupCodesPage`, which wrap `layout.Auth`. The swapped document nested a second brand mark under the original auth shell, so a wrong verification code showed two Flowbot logos.

## Decision

Match login / setup / login-2FA: GET keeps the auth layout; HTMX POST returns only the fragment that replaces the form (`Enroll2FAForm` on error, `BackupCodesPanel` after a valid code). `totp-enroll.js` already re-renders the QR on `htmx:afterSwap`.

## Alternatives considered

- **Detect `HX-Request` and return a full page for non-HTMX POSTs.** Rejected: login and setup already always return fragments; a second contract is extra surface without a noscript enroll path.
- **Change the form to `hx-target="body"`.** Rejected: that still ships a full document on every failed attempt and diverges from the other auth forms.

## Consequences

- A failed enroll POST is a form fragment, not a document. Clients that expected a full HTML page on that POST will only receive the form.
- After a valid code, backup codes replace the enroll form in the existing auth shell; `GET /service/web/setup/backup-codes` remains the ack page (codes are shown once).

## Verification

- `TestEnroll2FAHTMXFragments` asserts GET includes the auth brand and doctype; invalid and valid POSTs do not.
