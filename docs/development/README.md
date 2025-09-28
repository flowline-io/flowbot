# Development Documentation

This directory contains development-related tools and configurations for FlowBot.

## File Descriptions

### `example.fish`

Fish Shell script example containing common development commands and workflows.

### `http-client.private.env.json`

Environment configuration file for HTTP clients, used with VS Code REST Client or IntelliJ HTTP Client.

## Development Environment Setup

### Required Tools

1. **Go Environment**

```bash
# Install Go 1.19+
go version
```

2. **Task Tool**

```bash
# Install Task (task runner)
go install github.com/go-task/task/v3/cmd/task@latest

# View available tasks
task -a
```

3. **Code Quality Tools**

```bash
# Install code linting tool
go install github.com/mgechev/revive@latest

# Install security checking tool
go install golang.org/x/vuln/cmd/govulncheck@latest
```

4. **API Documentation Tool**

```bash
# Install Swagger generation tool
go install github.com/swaggo/swag/cmd/swag@latest
```

### Development Workflow

#### 1. Code Generation

```bash
# Generate new Bot
go run github.com/flowline-io/flowbot/cmd/composer generator bot -name example -rule collect,command,cron,form,input,instruct

# Generate new Vendor
go run github.com/flowline-io/flowbot/cmd/composer generator vendor -name example
```

#### 2. Database Management

```bash
# Import migration
go run github.com/flowline-io/flowbot/cmd/composer migrate import

# Create new migration file
go run github.com/flowline-io/flowbot/cmd/composer migrate migration -name your_migration_name
```

#### 3. Workflow Management

```bash
# Import workflow configuration
go run github.com/flowline-io/flowbot/cmd/composer workflow import -token xxx -path ./docs/config/examples/docker_example.yaml
```

#### 4. Code Checking

```bash
# Run code check
revive -formatter friendly ./...

# Security vulnerability check
govulncheck ./...

# Code statistics
cloc --exclude-dir=node_modules --exclude-ext=json .
```

#### 5. API Documentation

```bash
# Generate API documentation
swag init -g cmd/main.go

# Format API comments
swag fmt -g cmd/main.go
```

### HTTP Client Usage

After installing the REST Client extension in VS Code, you can use the `docs/api/api.http` file to test APIs.

Environment variables are configured in `http-client.private.env.json`:

```json
{
  "dev": {
    "baseUrl": "http://localhost:8080",
    "token": "your-dev-token"
  },
  "prod": {
    "baseUrl": "https://api.example.com",
    "token": "your-prod-token"
  }
}
```

### Debugging Tips

1. **Enable verbose logging**

```bash
export FLOWBOT_LOG_LEVEL=debug
```

2. **View runtime information**
   Visit `/dev/example` endpoint to get system information

3. **Get stack trace**
   Visit `/server/stacktrace` endpoint to get Goroutine information

### Contributing Guidelines

1. Fork the project
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add some amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Create a Pull Request
