# FlowBot Documentation

FlowBot is a workflow-based intelligent chatbot framework.

## Directory Structure

- 📁 [`api/`](./api/) - API documentation and interface definitions

  - `swagger.json` - OpenAPI 3.0 specification file (JSON format)
  - `swagger.yaml` - OpenAPI 3.0 specification file (YAML format)
  - `docs.go` - Auto-generated API documentation code
  - `api.http` - HTTP request examples collection

- 📁 [`config/`](./config/) - Configuration files and examples

  - `config.yaml` - Main configuration file template
  - `agent.yaml` - Agent configuration file template
  - [`examples/`](./config/examples/) - Configuration example files

- 📁 [`deployment/`](./deployment/) - Deployment-related documentation

  - `flowbot-agent.service` - Systemd service configuration file

- 📁 [`development/`](./development/) - Development-related documentation and tools

  - `example.fish` - Fish Shell script example
  - `http-client.private.env.json` - HTTP client environment configuration

- 📁 [`database/`](./database/) - Database-related documentation

  - `schema.md` - Database table structure documentation

- 📁 [`architecture/`](./architecture/) - System architecture documentation

  - `architecture.png` - System architecture diagram
  - `flowchart.mermaid` - Workflow flowchart

- 📄 `notify.md` - Notification configuration guide

## Quick Start

1. **Configuration**: Refer to configuration files in the [`config/`](./config/) directory
2. **Deployment**: Check deployment guides in the [`deployment/`](./deployment/) directory
3. **API**: View API documentation in the [`api/`](./api/) directory
4. **Development**: Refer to development tools in the [`development/`](./development/) directory

## Development Tools

### Task Management

```shell
# Install
go install github.com/go-task/task/v3/cmd/task@latest

# View available tasks
task -a
```

### Code Generation

```shell
# Generate bot
go run github.com/flowline-io/flowbot/cmd/composer generator bot -name example -rule collect,command,cron,form,input,instruct

# Generate vendor
go run github.com/flowline-io/flowbot/cmd/composer generator vendor -name example
```

### Database Migration

```shell
# Import migration
go run github.com/flowline-io/flowbot/cmd/composer migrate import

# Create migration file
go run github.com/flowline-io/flowbot/cmd/composer migrate migration -name file_name

# Import workflow
go run github.com/flowline-io/flowbot/cmd/composer workflow import -token xxx -path ./docs/config/examples/docker_example.yaml
```

### Code Linting

```shell
# Install revive
go install github.com/mgechev/revive@latest

# Run code check
revive -formatter friendly ./...
```

### Code Statistics

```shell
# Install cloc
sudo apt install cloc  # Linux
brew install cloc      # macOS

# Count code
cloc --exclude-dir=node_modules --exclude-ext=json .
```

### Security Check

```shell
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run security check
govulncheck ./...
```

### API Documentation Generation

> Reference: https://github.com/swaggo/swag/blob/master/README.md

```shell
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/main.go

# Format documentation
swag fmt -g cmd/main.go
```

### Database Migration Tool

```shell
# Install migration tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migration
migrate -source file://./internal/store/migrate -database mysql://user:password@tcp(127.0.0.1:3306)/db?parseTime=True&collation=utf8mb4_unicode_ci up
```

### Git Leak Detection

```shell
# Install gitleaks
go install github.com/zricethezav/gitleaks/v8@v8.21.1

# Check for leaks
gitleaks git -v
```

## Contributing

1. Fork this project
2. Create a feature branch
3. Commit your changes
4. Create a Pull Request

## License

This project is licensed under the GPL 3.0 License. See the [LICENSE](../LICENSE) file for details.
