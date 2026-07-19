# Database Documentation

Flowbot uses PostgreSQL as the primary database. Models are defined as Ent schemas (see `internal/store/ent/schema/`). Ent maps each schema file to a table.

## Schema Reference

The full generated column reference is in [`schema.md`](./schema.md). It is a point-in-time dump and may lag behind the schemas; treat `internal/store/ent/schema/` as the source of truth.

## Table Categories

Tables are grouped by responsibility below. Each row corresponds to one Ent schema file in `internal/store/ent/schema/`.

### Users and Authentication

- `users` — User accounts
- `oauth` — OAuth authentication tokens
- `authentication` — Auxiliary authentication records
- `topics` — Context/tenant management

### Platform Integration

- `platforms` — Registered chat platforms
- `platform_users` — Platform user mappings
- `platform_channels` — Platform channel mappings
- `platform_channel_users` — Channel-user associations
- `platform_bots` — Platform bot registrations

### Bot System

- `bots` — Bot definitions
- `agents` — Desktop agent records
- `agent_skills` — Agent skill registrations

### Messaging

- `messages` — Message records
- `channels` — Channel management
- `chat_sessions` — Agent chat session state
- `chat_session_entries` — Agent chat session messages/turns

### Hub and Homelab

- `apps` — Homelab scanned apps
- `capability_bindings` — Capability-to-backend bindings
- `connections` — Hub connection records

### Pipeline System

- `pipeline_definitions` — Pipeline definition records
- `pipeline_definition_versions` — Versioned pipeline definition history
- `pipeline_runs` — Pipeline execution runs
- `pipeline_step_runs` — Pipeline step execution records
- `event_consumptions` — Pipeline idempotency guard

### Workflow System

- `workflow_runs` — Workflow execution runs
- `workflow_step_runs` — Workflow step execution records

### Events

- `data_events` — Durable business events
- `event_outbox` — Transactional outbox for event publishing
- `polling_state` — Per-provider polling cursor state

### Notifications

- `notify_channels` — Per-user notification channel configuration
- `notify_rules` — Notification gateway rules
- `notification_records` — Notification delivery history

### Resources

- `resource_links` — Tag/chain links between resources

### Data Storage

- `configs` — Key-value configuration storage
- `data` — General key-value data storage
- `form` — Form schemas and submissions
- `pages` — Page configurations
- `page_data` — Page data payloads
- `parameter` — Temporary parameter storage
- `instruct` — Instruction records
- `urls` — URL tracking
- `file_uploads` — File upload records

### Analytics

- `behavior` — User behavior statistics
- `counters` / `counter_records` — Counter system

### Audit

- `audit_logs` — Audit log entries

## Database Schema Management

Ent auto-migration via `client.Schema.Create()` on startup. No manual SQL migrations.

## Code Generation

```bash
go tool task ent     # Generate ent code from schemas
go tool task templ   # Generate templ Go code
```

> Note: the legacy `task doc` schema-documentation command has been removed. `schema.md` is the last generated snapshot and is no longer refreshed automatically.

## Configuration

```yaml
postgres:
  dsn: "postgres://user:password@localhost:5432/flowbot?sslmode=disable"
```

## Backup

```bash
pg_dump -U user flowbot > backup_$(date +%Y%m%d_%H%M%S).sql
psql -U user flowbot < backup_file.sql
```
