# Flowbot Documentation

Flowbot is an advanced multi-platform chatbot framework that provides intelligent conversation, workflow automation, and comprehensive LLM agent capabilities.

## Directory Structure

- 📁 [`api/`](./api/) - API documentation and interface definitions
  - `swagger.json` - OpenAPI 3.0 specification file (JSON format)
  - `swagger.yaml` - OpenAPI 3.0 specification file (YAML format)
  - `docs.go` - Auto-generated API documentation code
  - `api.http` - HTTP request examples collection

- 📁 [`config/`](./config/) - Configuration files and examples
  - `config.yaml` - Main configuration file template
  - `agent.yaml` - Agent configuration file template
  - [`examples/`](./config/examples/) - Workflow configuration examples

- 📁 [`deployment/`](./deployment/) - Deployment-related documentation
  - `flowbot-agent.service` - Systemd service configuration file

- 📁 [`database/`](./database/) - Database-related documentation
  - `schema.md` - Database table structure documentation

- 📁 [`architecture/`](./architecture/) - System architecture documentation
  - `architecture.png` - System architecture diagram
  - `flowchart.mermaid` - Workflow flowchart (Mermaid)

- 📄 [`notify.md`](./notify.md) - Notification configuration guide
- 📄 [`schema.md`](./schema.md) - Database schema reference

## Quick Start

1. **Configuration**: Refer to configuration files in the [`config/`](./config/) directory
2. **Deployment**: Check deployment guides in the [`deployment/`](./deployment/) directory
3. **API**: View API documentation in the [`api/`](./api/) directory
4. **Architecture**: Review system design in the [`architecture/`](./architecture/) directory

## Development Tools

All development tools are managed as Go tools (via `go get -tool`) or through [Task](https://taskfile.dev) runner.

### Task Runner

```shell
# View all available tasks
task -a

# Run common checks (tidy → swagger → format → lint → scc)
task default

# Build all binaries
task build:all
```

### Build Commands

```shell
# Build main server
task build

# Build agent
task build:agent

# Build admin PWA (Wasm + server)
task build:app

# Build composer CLI
task build:composer

# Run with live reload
task air
```

### Code Generation

```shell
# Generate bot scaffolding
task generator:bot NAME=example RULE=command,form

# Generate vendor API code
task generator:vendor NAME=example

# Generate DAO code from database
task dao
```

### Database Migration

```shell
# Import migrations
task migrate

# Create new migration file
task migration NAME=add_new_feature

# Import workflow configuration
task workflow:import TOKEN=xxx PATH=./docs/config/examples/docker_example.yaml
```

### Code Quality

```shell
# Lint (revive + actionlint)
task lint

# Format code (go fmt + prettier)
task format

# Tidy Go modules
task tidy
```

### Security

```shell
# Vulnerability check
task secure

# Secret leak detection
task leak

# Go security checker
task gosec

# Run all security & quality checks
task check
```

### Testing

```shell
# Run unit tests
task test

# Run all tests
task test:all

# Generate coverage report
task test:coverage
```

### API Documentation

```shell
# Generate Swagger docs
task swagger

# Generate database schema docs
task doc
```

### Add Go Tool Dependency

```shell
go get -tool import_path@version
```

## Contributing

1. Fork this project
2. Create a feature branch
3. Commit your changes
4. Create a Pull Request

## License

This project is licensed under the GPL 3.0 License. See the [LICENSE](../LICENSE) file for details.
