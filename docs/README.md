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
go tool task build

# Build agent
go tool task build:agent

# Build admin PWA (Wasm + server)
go tool task build:app

# Build composer CLI
go tool task build:composer

# Run with live reload
go tool task air
```

### Code Generation

```shell
# Generate DAO code from database
go tool task dao

# Generate Swagger/OpenAPI docs
go tool task swagger

# Generate database schema docs
go tool task doc
```

### Database Migration

Migrations run automatically at server startup. See `pkg/migrate/`.

### Workflow CLI

```shell
# Run a workflow YAML file
go run ./cmd/cli workflow run ./docs/config/examples/docker_example.yaml
```

### Code Quality

```shell
# Lint (revive + actionlint)
go tool task lint

# Format code (go fmt + prettier)
go tool task format

# Tidy Go modules
go tool task tidy
```

### Security

```shell
# Vulnerability check
go tool task secure

# Secret leak detection
go tool task leak

# Go security checker
go tool task gosec

# Run all security & quality checks
go tool task check
```

### Testing

```shell
# Run unit tests
go tool task test

# Run all tests
go tool task test:all

# Generate coverage report
go tool task test:coverage
```

### API Documentation

```shell
# Generate Swagger docs
go tool task swagger

# Generate database schema docs
go tool task doc
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
