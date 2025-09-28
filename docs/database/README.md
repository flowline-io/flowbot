# Database Documentation

This directory contains database-related documentation for FlowBot.

## File Descriptions

### `schema.md`

Complete database table structure documentation, including field definitions, indexes, and constraints for all tables.

## Database Design Overview

FlowBot uses relational databases (supports MySQL/PostgreSQL) to store application data.

### Core Table Structure

#### Users and Permissions

- `users` - User basic information
- `oauth` - OAuth authentication information
- `topics` - Topics/tenant management

#### Platform Integration

- `platforms` - Supported chat platforms
- `platform_users` - Platform user information
- `platform_channels` - Platform channel information
- `platform_bots` - Platform bot associations

#### Workflow System

- `workflow` - Workflow definitions
- `workflow_script` - Workflow scripts
- `workflow_trigger` - Workflow triggers
- `jobs` - Workflow execution tasks
- `steps` - Task execution steps
- `dag` - Directed Acyclic Graph definitions

#### OKR Management

- `objectives` - Objective management
- `key_results` - Key results
- `key_result_values` - Key result values
- `reviews` - Review records
- `todos` - Todo items

#### Messaging and Communication

- `messages` - Message records
- `channels` - Channel management
- `bots` - Bot definitions

#### System Functions

- `configs` - Configuration storage
- `data` - General data storage
- `form` - Form definitions and data
- `pages` - Page configuration
- `session` - Session management
- `behavior` - Behavior statistics
- `counters` - Counters

## Database Migration

### Migration Tool Installation

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Execute Migration

```bash
# MySQL
migrate -source file://./internal/store/migrate \
  -database "mysql://user:password@tcp(localhost:3306)/flowbot?parseTime=True&collation=utf8mb4_unicode_ci" \
  up

# PostgreSQL
migrate -source file://./internal/store/migrate \
  -database "postgres://user:password@localhost:5432/flowbot?sslmode=disable" \
  up
```

### Create New Migration

```bash
# Use composer tool to create migration
go run github.com/flowline-io/flowbot/cmd/composer migrate migration -name add_new_feature
```

## Database Configuration

### MySQL Configuration Example

```yaml
database:
  type: mysql
  host: localhost
  port: 3306
  name: flowbot
  user: flowbot_user
  password: your_password
  charset: utf8mb4
  collation: utf8mb4_unicode_ci
```

### PostgreSQL Configuration Example

```yaml
database:
  type: postgres
  host: localhost
  port: 5432
  name: flowbot
  user: flowbot_user
  password: your_password
  sslmode: disable
```

## Performance Optimization Recommendations

### Index Optimization

- Ensure all foreign key fields have indexes
- Add composite indexes for frequently queried fields
- Regularly analyze query performance

### Connection Pool Configuration

```yaml
database:
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m
```

### Monitoring Metrics

- Connection usage
- Slow query logs
- Table space usage
- Index usage statistics

## Backup Strategy

### Regular Backup

```bash
# MySQL backup
mysqldump -u user -p flowbot > backup_$(date +%Y%m%d_%H%M%S).sql

# PostgreSQL backup
pg_dump -U user -h localhost flowbot > backup_$(date +%Y%m%d_%H%M%S).sql
```

### Data Restoration

```bash
# MySQL restoration
mysql -u user -p flowbot < backup_file.sql

# PostgreSQL restoration
psql -U user -h localhost flowbot < backup_file.sql
```
