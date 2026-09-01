# Agent Note: Endpoint health checker uses a private HTTP transport

Status: implemented

## Problem

`TestEndpointHealthChecker_Check/2xx_is_healthy` failed in CI with `net/http: HTTP/1.x transport connection broken: http: CloseIdleConnections called`. `NewEndpointHealthChecker` built `&http.Client{Timeout: timeout}`, which uses `http.DefaultTransport`. `httptest.Server.Close` always calls `DefaultTransport.CloseIdleConnections`, so a parallel subtest finishing its server tore down keep-alives (and in-progress dials) for an in-flight probe on another server.

## Decision

`NewEndpointHealthChecker` clones `http.DefaultTransport` (or allocates a fresh `http.Transport`) so probes do not share the default idle pool. `httptest.Server.Close` still closes DefaultTransport; it no longer observes checker connections.

## Alternatives considered

- **Drop `t.Parallel()` on the Check table.** Would hide the flake in this package only; production probes would still share DefaultTransport with anything that calls `CloseIdleConnections`.
- **Use `utils.HTTPTransport()`.** Same isolation from DefaultTransport, but `pkg/hub` would import the resty/otel utils package for a clone it can do locally.
- **Disable keep-alives on the default client.** Removes the idle-pool race, not `CloseIdleConnections` canceling DefaultTransport dials still in progress.

## Consequences

Each checker has its own connection pool. The shared `defaultEndpointHealthChecker` still reuses one `http.Client` across `/healthz` calls. Tests that close `httptest.Server` in parallel no longer poison endpoint probes.

## Verification

`TestEndpointHealthChecker_Check_IgnoresDefaultTransportCloseIdle` in `pkg/hub/endpoint_health_test.go` requires a non-nil private Transport and a successful probe after `DefaultTransport.CloseIdleConnections`. `TestEndpointHealthChecker_Check` remains the original parallel httptest table.
