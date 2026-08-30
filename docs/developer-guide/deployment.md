# Deployment Documentation

## Build

All binaries are built using [Task](https://taskfile.dev):

```bash
task build           # Main server (bin/flowbot)
task build:composer  # Composer CLI (bin/composer)
task build:cli       # Admin CLI (bin/flowbot-cli)
task build:cli:linux # linux/amd64 CLI for sandbox inject (bin/flowbot-cli_linux_amd64)
task build:all       # All binaries
```

## Deployment Methods

### 1. Binary Deployment

```bash
task build
./bin/flowbot                      # Start server
./bin/flowbot-cli -- server-url http://localhost:6060  # Admin CLI
# For chatagent Docker sandbox skill→CLI: place linux CLI beside the server binary, e.g.
#   task build:cli:linux && copy bin/flowbot-cli_linux_amd64 next to bin/flowbot
# or set chat_agent.sandbox.cli_path.
```

### 2. Docker Deployment

```bash
docker build -f deployments/Dockerfile -t flowbot .
docker run -p 6060:6060 -v $(pwd)/flowbot.yaml:/opt/app/flowbot.yaml flowbot
```

The server image installs `dcg` on `PATH` (`/usr/local/bin/dcg`, linux musl amd64, SHA256-verified) and ships [`pkg/agent/dcg/config.toml`](../../pkg/agent/dcg/config.toml) at `/etc/dcg/config.toml` for Always-on chat-agent `run_terminal` / `run_code` guards. Flowbot still materializes the embedded config at runtime; the image file is for operational parity. The image also ships `flowbot-cli_linux_amd64` beside the server binary so chatagent sandbox can inject it into ephemeral containers (see [Agent Sandbox](../agent/agent-sandbox.md)).

For the Cloud Agent ephemeral sandbox image (`flowbot-agent-sandbox`), see [Agent Sandbox](../agent/agent-sandbox.md).

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
| `docker-agent-sandbox.yml` | Build agent sandbox image (`sandbox-v*` tags) |
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
- [ ] Orphan tables dropped after schema removals (see [Database schema upgrades](#database-schema-upgrades))

## Database schema upgrades

Ent auto-migration (`Schema.Create()` on startup) creates and alters tables but **does not drop** tables removed from the schema. After upgrading to a build that removed unused Ent entities, run the following on PostgreSQL once (safe when tables are empty or data is abandoned):

```sql
DROP TABLE IF EXISTS authentications;
DROP TABLE IF EXISTS connections;
DROP TABLE IF EXISTS capability_bindings;
DROP TABLE IF EXISTS platform_bots;
DROP TABLE IF EXISTS topics;
DROP TABLE IF EXISTS urls;
```

Rationale and replacements: [Agent Note: Drop unused Ent tables](../../.agents/notes/implemented/simplification/2026-08-28-drop-unused-ent-tables.md).

Verify before drop:

```sql
SELECT 'topics' AS t, COUNT(*) FROM topics
UNION ALL SELECT 'urls', COUNT(*) FROM urls
UNION ALL SELECT 'connections', COUNT(*) FROM connections
UNION ALL SELECT 'authentications', COUNT(*) FROM authentications
UNION ALL SELECT 'platform_bots', COUNT(*) FROM platform_bots
UNION ALL SELECT 'capability_bindings', COUNT(*) FROM capability_bindings;
```
