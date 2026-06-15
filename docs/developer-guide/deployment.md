# Deployment Documentation

## Build

All binaries are built using [Task](https://taskfile.dev):

```bash
task build           # Main server (bin/flowbot)
task build:composer  # Composer CLI (bin/composer)
task build:cli       # Admin CLI (bin/flowbot-cli)
task build:all       # All binaries
```

## Deployment Methods

### 1. Binary Deployment

```bash
task build
./bin/flowbot                      # Start server
./bin/flowbot-cli -- server-url http://localhost:6060  # Admin CLI
```

### 2. Docker Deployment

```bash
docker build -f deployments/Dockerfile -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot
```

### 3. Systemd Service

For headless Linux deployments, run the main server under systemd:

1. Copy binary and service file:

```bash
sudo cp bin/flowbot /opt/app/
sudo chmod +x /opt/app/flowbot
sudo cp docs/developer-guide/flowbot.service /etc/systemd/system/
```

2. Place your runtime configuration and edit it for your environment:

```bash
sudo cp docs/reference/config.yaml /opt/app/flowbot.yaml
```

3. Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable flowbot
sudo systemctl start flowbot
```

#### Service Management

```bash
sudo systemctl status flowbot
sudo systemctl restart flowbot
sudo journalctl -u flowbot -f
```

## CI/CD

GitHub Actions workflows (`.github/workflows/`):

| Workflow        | Description        |
| --------------- | ------------------ |
| `build.yml`     | Lint + Build       |
| `testing.yml`   | Run all tests      |
| `build_cli.yml` | Build CLI tools    |
| `docker.yml`    | Build Docker image |
| `release.yml`   | Release pipeline   |

## Health Checks

```bash
curl http://localhost:6060/livez    # Liveness
curl http://localhost:6060/readyz   # Readiness
curl http://localhost:6060/startupz # Startup
```

## Deployment Checklist

- [ ] Configuration file (`flowbot.yaml`) is set up
- [ ] PostgreSQL database is accessible
- [ ] Redis server is running
- [ ] Required ports are open (default: 6060)
- [ ] Service starts and health checks pass
