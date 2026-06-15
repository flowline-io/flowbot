# Developer Guide

Operations and development documentation for Flowbot.

## Contents

- [Deployment](./deployment.md) — Binary, Docker, and systemd deployment methods with health checks
- [Monitoring](./monitoring.md) — Grafana dashboard, Prometheus metrics via PushGateway, and alerting rules
- [Tracing](./tracing.md) — OpenTelemetry distributed tracing across all components
- [Conformance](./conformance.md) — Ability adapter conformance test suite for provider development

## Development Tools

### Build

```bash
go tool task build           # Main server
go tool task build:composer  # Composer CLI
go tool task build:cli       # Admin CLI
go tool task build:chat      # Chat Agent terminal client
go tool task build:all       # All binaries
go tool task air             # Live reload
```

### Code Generation

```bash
go tool task swagger   # Generate Swagger/OpenAPI docs
go tool task skills    # Generate SKILL.md files for CLI capabilities
go tool task webdoc    # Generate website docs from markdown sources
go tool task ent       # Generate ent code from database schema
go tool task templ     # Generate Go code from Templ templates
```

### Code Quality

```bash
go tool task lint      # revive + actionlint + testifylint + oxlint
go tool task format    # go fmt + oxfmt
go tool task tidy      # go mod tidy
```

### Security

```bash
go tool task secure    # govulncheck
go tool task leak      # gitleaks
go tool task gosec     # security scan
go tool task check     # all security & quality
```

### Testing

```bash
go tool task test            # All unit tests
go tool task test:short      # Short mode (skip integration)
go tool task test:specs      # BDD acceptance tests (requires Docker)
go tool task test:coverage   # Coverage report
```

### Conformance Tests

```bash
go test ./pkg/ability/...                                # All ability + conformance tests
go test -run TestConformance ./pkg/ability/bookmark/karakeep/  # Single adapter
go test ./pkg/ability/conformance/                       # Conformance framework self-tests
```

### Add Go Tool Dependency

```bash
go get -tool import_path@version
```

## systemd Service

The [flowbot.service](./flowbot.service) file is provided for headless Linux deployments.
