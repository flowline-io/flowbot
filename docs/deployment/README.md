# Deployment Documentation

This directory contains deployment-related documentation and configuration files for Flowbot.

## File Descriptions

### `flowbot-agent.service`

Systemd service configuration file for running Flowbot Agent as a system service on Linux.

## Build

All binaries are built using [Task](https://taskfile.dev):

```bash
# Build main server
task build

# Build agent
task build:agent

# Build admin PWA (Wasm + server)
task build:app

# Build all
task build:all
```

## Deployment Methods

### 1. Docker Deployment (Recommended)

Two Dockerfiles are available in the `deployments/` directory:

| File | Image | Description |
|------|-------|-------------|
| `Dockerfile` | `flowbot` | Main server (single binary) |
| `Dockerfile.app` | `flowbot-app` | Admin PWA (multi-stage: Wasm + server on Alpine) |

#### Build Docker Images

```bash
# Main server
docker build -f deployments/Dockerfile -t flowbot .

# Admin PWA
docker build -f deployments/Dockerfile.app -t flowbot-app .
```

#### Run

```bash
# Main server
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot

# Admin PWA
docker run -p 8090:8090 flowbot-app
```

### 2. Systemd Service Deployment

#### Agent Service

1. Copy binary and service file:

```bash
sudo cp bin/flowbot-agent /opt/app/
sudo chmod +x /opt/app/flowbot-agent
sudo cp docs/deployment/flowbot-agent.service /etc/systemd/system/
```

2. Create environment file:

```bash
sudo vi /opt/app/flowbot-agent.env
```

3. Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable flowbot-agent
sudo systemctl start flowbot-agent
```

#### Service Management

```bash
sudo systemctl status flowbot-agent
sudo systemctl restart flowbot-agent
sudo journalctl -u flowbot-agent -f
```

### 3. Manual Deployment

```bash
# Server
./bin/flowbot

# Agent
./bin/flowbot-agent

# Admin PWA
./bin/flowbot-app
```

## CI/CD

GitHub Actions workflows (`.github/workflows/`):

| Workflow | Description |
|----------|-------------|
| `build.yml` | Build main server |
| `build_agent.yml` | Build agent |
| `build_app.yml` | Build admin PWA + Docker image (uses `task build:app`) |
| `docker.yml` | Docker image publishing |
| `release.yml` | Release pipeline |

## Health Checks

The main server exposes health check endpoints:

```bash
curl http://localhost:6060/livez    # Liveness
curl http://localhost:6060/readyz   # Readiness
curl http://localhost:6060/startupz # Startup
```

## Deployment Checklist

- [ ] Configuration file (`flowbot.yaml`) is set up
- [ ] MySQL database is accessible
- [ ] Redis server is running
- [ ] Required ports are open (default: 6060 for server, 8090 for PWA)
- [ ] Log directory has write permissions
- [ ] Service starts and health checks pass
