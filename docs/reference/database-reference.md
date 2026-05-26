# Database Documentation

Flowbot uses PostgreSQL as the primary database. Models are defined as Ent schemas (see `internal/store/ent/schema/`).

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

### Pipeline System

- `pipeline_definitions` — Pipeline definition records
- `pipeline_runs` — Pipeline execution runs
- `pipeline_step_runs` — Pipeline step execution records

### Analytics

- `behavior` — User behavior statistics
- `counters` / `counter_records` — Counter system

### Other

- `urls` — URL tracking
- `fileuploads` — File upload records

## Database Schema Management

Ent auto-migration via `client.Schema.Create()` on startup. No manual SQL migrations.

## Code Generation

```bash
task doc       # Generate schema documentation
```

## Configuration

```yaml
store_config:
  use_adapter: postgres
  adapters:
    postgres:
      dsn: "postgres://user:password@localhost:5432/flowbot?sslmode=disable"
```

## Backup

```bash
pg_dump -U user flowbot > backup_$(date +%Y%m%d_%H%M%S).sql
psql -U user flowbot < backup_file.sql
```
