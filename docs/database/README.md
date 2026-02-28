# Database Documentation

This directory contains database-related documentation for Flowbot.

## File Descriptions

### `schema.md`

Complete database table structure documentation, including field definitions, indexes, and constraints for all tables.

## Database Design Overview

Flowbot uses MySQL as the primary database. All models are auto-generated via GORM Gen (see `internal/store/model/`).

### Core Tables

#### Users and Authentication

- `users` - User basic information
- `oauth` - OAuth authentication records
- `topics` - Topics/tenant management

#### Platform Integration

- `platforms` - Registered chat platforms
- `platform_users` - Platform user mappings
- `platform_channels` - Platform channel mappings
- `platform_channel_users` - Channel-user associations
- `platform_bots` - Platform bot registrations

#### Workflow System

- `workflow` - Workflow definitions
- `workflow_script` - Workflow scripts (versioned)
- `workflow_trigger` - Workflow triggers (manual, webhook, cron)
- `jobs` - Workflow execution jobs
- `steps` - Job execution steps
- `dag` - Directed Acyclic Graph definitions

#### Flow Engine (v2)

- `flows` - Flow definitions
- `flow_nodes` - Flow node definitions
- `flow_edges` - Flow edge connections
- `flow_jobs` - Flow job records
- `flow_queue_jobs` - Flow job queue
- `executions` - Flow execution records
- `connections` - External service connections
- `authentications` - Connection authentication data

#### Bot System

- `bots` - Bot definitions
- `agents` - Desktop agent records
- `apps` - Application registrations
- `webhook` - Webhook configurations

#### Messaging

- `messages` - Message records (with role and session)
- `channels` - Channel management

#### OKR System

- `objectives` - Objectives
- `key_results` - Key results
- `key_result_values` - Key result value tracking
- `reviews` / `review_evaluations` - Review records
- `cycles` - OKR cycles
- `todos` - Todo items

#### Data Storage

- `configs` - Key-value configuration storage
- `data` - General key-value data storage
- `form` - Form schemas and submissions
- `pages` - Page configurations
- `parameter` - Temporary parameter storage
- `instruct` - Instruction records

#### Analytics

- `behavior` - User behavior statistics
- `counters` / `counter_records` - Counter system
- `rate_limits` - API rate limiting

#### Other

- `urls` - URL tracking
- `fileuploads` - File upload records
- `schema_migrations` - Migration version tracking

## Database Migration

Migrations are managed via the Composer CLI and stored in `internal/store/migrate/migrations/` (currently 51 migration files).

### Run Migrations

```bash
# Import all migrations
task migrate

# Or use composer directly
go run ./cmd/composer migrate import
```

### Create New Migration

```bash
# Via task
task migration NAME=add_new_feature

# Or directly
go run ./cmd/composer migrate migration -name add_new_feature
```

### Generate Schema Documentation

```bash
task doc
```

### Generate DAO Code

After modifying the database schema, regenerate the DAO code:

```bash
task dao
```

## Database Configuration

### MySQL Configuration (in `flowbot.yaml`)

```yaml
store_config:
  use_adapter: mysql
  adapters:
    mysql:
      dsn: "user:password@tcp(localhost:3306)/flowbot?parseTime=True&collation=utf8mb4_unicode_ci"
```

## Backup

```bash
# MySQL backup
mysqldump -u user -p flowbot > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore
mysql -u user -p flowbot < backup_file.sql
```
