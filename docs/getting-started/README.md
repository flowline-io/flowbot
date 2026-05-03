# Getting Started

Quick start guide for installing and configuring Flowbot.

## Installation

```bash
git clone https://github.com/anomalyco/flowbot
cd flowbot
go tool task build
```

For detailed deployment options (binary, Docker, systemd), see [Developer Guide / Deployment](../developer-guide/deployment.md).

## Configuration

Copy the configuration template and edit it for your environment:

```bash
cp docs/reference/config.yaml flowbot.yaml
```

See [Reference / Configuration](../reference/config-reference.md) for a description of all configuration sections.

### Required Services

- **MySQL** — database (auto-migrates on startup)
- **Redis** — event streams and caching

### Quick Start

```bash
go tool task run    # Start the server
```

### Health Checks

```bash
curl http://localhost:6060/livez    # Liveness
curl http://localhost:6060/readyz   # Readiness
curl http://localhost:6060/startupz # Startup
```

## CLI Tools

```bash
go run ./cmd/cli workflow run ./docs/examples/workflows/save_and_track.yaml
```
