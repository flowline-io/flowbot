# Database Documentation

Flowbot uses MySQL as the primary database. Models are auto-generated via GORM Gen (see `internal/store/`).

## Schema Reference

Full table schema is in [`schema.md`](./schema.md).

## Table Categories

### Users and Authentication

- `users` — User accounts
- `oauth` — OAuth authentication tokens
- `topics` — Context/tenant management

### Platform Integration

- `platforms` — Registered chat platforms
- `platform_users` — Platform user mappings
- `platform_channels` — Platform channel mappings
- `platform_channel_users` — Channel-user associations
- `platform_bots` — Platform bot registrations

### Workflow System

- `workflow` — Workflow definitions
- `workflow_script` — Versioned workflow scripts
- `workflow_trigger` — Triggers (manual, webhook, cron)
- `jobs` — Execution jobs
- `steps` — Job execution steps
- `dag` — DAG definitions

### Bot System

- `bots` — Bot definitions
- `agents` — Desktop agent records
- `webhook` — Webhook configurations

### Messaging

- `messages` — Message records
- `channels` — Channel management

### OKR System

- `objectives` — Objectives
- `key_results` — Key results
- `key_result_values` — Key result value tracking
- `reviews` / `review_evaluations` — Review records
- `cycles` — OKR cycles
- `todos` — Todo items

### Data Storage

- `configs` — Key-value configuration storage
- `data` — General key-value data storage
- `form` — Form schemas and submissions
- `pages` — Page configurations
- `parameter` — Temporary parameter storage
- `instruct` — Instruction records

### Pipeline & Session

- `pipelines` — Pipeline execution records
- `session` — Pipeline session state

### Analytics

- `behavior` — User behavior statistics
- `counters` / `counter_records` — Counter system
- `action` — Action tracking

### Other

- `urls` — URL tracking
- `fileuploads` — File upload records
- `schema_migrations` — Migration version tracking

## Database Migration

Migrations run automatically at server startup via `pkg/migrate/`.

## Code Generation

```bash
task dao       # Generate DAO code from database
task doc       # Generate schema documentation
```

## Configuration

```yaml
store_config:
  use_adapter: mysql
  adapters:
    mysql:
      dsn: "user:password@tcp(localhost:3306)/flowbot?parseTime=True&collation=utf8mb4_unicode_ci"
```

## Backup

```bash
mysqldump -u user -p flowbot > backup_$(date +%Y%m%d_%H%M%S).sql
mysql -u user -p flowbot < backup_file.sql
```
