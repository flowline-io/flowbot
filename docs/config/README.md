# Configuration Files

This directory contains configuration templates and examples for Flowbot.

## File Descriptions

### `config.yaml`

Main application configuration file template for the Flowbot server. Covers:

- Server listen address and API path
- Media storage (file system / MinIO)
- Database connection (MySQL)
- Redis connection
- Logging configuration
- Executor settings (Docker / Shell / Machine)
- Prometheus metrics
- MeiliSearch integration
- LLM model configuration (OpenAI-compatible)
- AI agent definitions
- Platform integrations (Slack, Discord, Tailchat)
- Bot module settings
- Third-party vendor configurations

### `agent.yaml`

Dedicated configuration for the Flowbot Agent (`cmd/agent`). Settings include:

- Log level
- Enabled bot modules
- API connection (URL + token)
- GitHub updater token
- Script engine (paths, UID/GID, watch exclusions)
- Prometheus metrics endpoint

### `examples/`

Workflow configuration examples:

- `docker_example.yaml` - Docker container execution workflow
- `example.yaml` - Rule chain with JS filter/transform/log
- `func_example.yaml` - Custom function chain workflow

## Quick Start

1. Copy the appropriate template:
   ```bash
   # Server configuration
   cp docs/config/config.yaml flowbot.yaml

   # Agent configuration
   cp docs/config/agent.yaml flowbot-agent.yaml
   ```

2. Edit configuration values for your environment

3. Start the service:
   ```bash
   # Server
   task run

   # Agent
   task run:agent

   # Admin PWA
   task run:app
   ```

## Environment Variables

Configuration values can be overridden via environment variables:

```bash
export FLOWBOT_CONFIG_PATH=/path/to/flowbot.yaml
export FLOWBOT_LOG_LEVEL=info
```
