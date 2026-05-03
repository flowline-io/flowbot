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
- Homelab app scanner and endpoint/auth discovery
- Platform integrations (Slack, Discord, Tailchat, Telegram)
- Bot module settings
- Third-party vendor configurations

For details on the homelab discovery system, see [Homelab App Discovery](../user-guide/homelab-discovery.md).

### Workflow Examples

See [examples/workflows/](../examples/workflows/) for workflow configuration examples.

## Quick Start

1. Copy the appropriate template:

   ```bash
   # Server configuration
   cp docs/config/config.yaml flowbot.yaml
   ```

2. Edit configuration values for your environment

3. Start the service:

   ```bash
   task run
   ```

## Environment Variables

Configuration values can be overridden via environment variables:

```bash
export FLOWBOT_CONFIG_PATH=/path/to/flowbot.yaml
export FLOWBOT_LOG_LEVEL=info
```
