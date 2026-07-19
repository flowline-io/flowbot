# Getting Started

Quick start guide for installing and configuring Flowbot.

## Installation

```bash
git clone https://github.com/flowline-io/flowbot
cd flowbot
go tool task build
```

For detailed deployment options (binary, Docker, systemd), see [Self-hosting](../self-hosting.md) and [Developer Guide / Deployment](../developer-guide/deployment.md).

## Configuration

Copy the configuration template and edit it for your environment:

```bash
cp docs/reference/config.yaml flowbot.yaml
```

See [Reference / Configuration](../reference/config-reference.md) for a description of all configuration sections.

### Required Services

- **PostgreSQL** — database (auto-migrates on startup)
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

## CLI

Install the `flowbot` CLI binary from GitHub releases:

```bash
curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.sh | bash
```

This installs the latest CLI to `/usr/local/bin/flowbot`.

### Install a specific version

```bash
curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.sh | bash -s -- --version v0.40
```

### Skip checksum verification

```bash
curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install.sh | bash -s -- --no-verify
```

### Install from source

```bash
go install github.com/flowline-io/flowbot/cmd/cli@latest
```

### Usage

```bash
flowbot --help
flowbot workflow run ./docs/examples/workflows/save_and_track.yaml
```
