# Configuration Files

This directory contains various configuration files and examples for FlowBot.

## File Descriptions

### `config.yaml`

Main application configuration file, includes:

- Database configuration
- Server settings
- Logging configuration
- External service integration configuration

### `agent.yaml`

Dedicated configuration file for Agent service, includes:

- Agent-specific settings
- Task execution configuration
- Scheduling and monitoring settings

### `examples/`

Configuration examples directory, contains:

- `docker_example.yaml` - Docker deployment configuration example
- `example.yaml` - Basic configuration example
- `func_example.yaml` - Function configuration example

## Configuration Structure

### Basic Configuration

```yaml
# Server configuration
server:
  port: 8080
  host: "0.0.0.0"

# Database configuration
database:
  type: mysql
  host: localhost
  port: 3306
  name: flowbot

# Logging configuration
log:
  level: info
  format: json
```

### Agent Configuration

```yaml
# Agent basic settings
agent:
  name: "flowbot-agent"
  interval: "30s"

# Task execution configuration
executor:
  workers: 4
  timeout: "5m"
```

## Usage

1. Copy example configuration files
2. Modify configuration parameters according to environment
3. Set necessary environment variables
4. Start the service
