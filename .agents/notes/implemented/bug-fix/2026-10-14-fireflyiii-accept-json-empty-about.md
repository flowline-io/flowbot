# Agent Note: Firefly III Accept JSON and empty about/user

Status: implemented

## Problem

After fixing the prod endpoint so flowbot could reach Firefly III, `flowbot fireflyiii health` reported healthy while `about` and `user` printed empty fields. An invalid Personal Access Token caused Firefly to redirect browser-style requests (no `Accept` header) to `/login` HTML with a final HTTP 200. The provider treated that as success with a nil `data` payload, so health (which only checks `About()` error) stayed green.

## Decision

- Set `Accept: application/json, application/vnd.api+json` on the Firefly III resty client so unauthenticated calls return JSON 401 with `message`/`exception` instead of an HTML login redirect.
- Make `ConvertResponseData` fail when `data` is nil on HTTP 200, so empty bodies cannot look like a healthy instance.

Ops still must keep a valid Firefly Personal Access Token in `providers.fireflyiii.token`.

## Alternatives considered

- Disable redirect following only: would surface errors for HTML redirects but still leave Accept ambiguous for other clients and middlewares; Accept is the contract Firefly documents.
- Keep silent empty about and only document the token requirement: health would keep lying; rejected.

## Consequences

- Invalid or missing tokens fail `about` / `user` / `health` with a provider error (HTTP 502 at the Flowbot API) instead of empty successful payloads.
- Callers that relied on empty about as “reachable” must treat health as false until the token is fixed.

## Verification

```bash
go test ./pkg/providers/fireflyiii/ -count=1
```

With a bad token and Accept set, About returns a non-nil error (401 message). With a valid token, about/user fields are populated.
