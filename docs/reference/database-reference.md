# Database Documentation

Flowbot uses PostgreSQL as the primary database. Models are defined as Ent schemas (see `internal/store/ent/schema/`). Ent maps each schema file to a table.

## Schema Reference

The full generated column reference is in [`schema.md`](./schema.md). It is a point-in-time dump and may lag behind the schemas; treat `internal/store/ent/schema/` as the source of truth.

## Table Categories

Tables are grouped by responsibility below. Each row corresponds to one Ent schema file in `internal/store/ent/schema/`.

### Users and Authentication

- `users` — User accounts
- `oauth` — OAuth authentication tokens

### Platform Integration

- `platforms` — Registered chat platforms
- `platform_users` — Platform user mappings
- `platform_channels` — Platform channel mappings
- `platform_channel_users` — Channel-user associations

### Bot System

- `bots` — Module / bot registration state

### Messaging

- `messages` — Message records
- `channels` — Channel management

### Hub and Homelab

- `apps` — Homelab scanned apps

### Pipeline System

- `pipeline_definitions` — Pipeline definition records
- `pipeline_definition_versions` — Versioned pipeline definition history
- `pipeline_name` — Published pipeline name index
- `pipeline_runs` — Pipeline execution runs
- `pipeline_step_runs` — Pipeline step execution records
- `event_consumptions` — Pipeline idempotency guard

### Workflow System

- `workflows` — Workflow definitions
- `workflow_tasks` — Workflow task nodes
- `workflow_triggers` — Workflow triggers
- `workflow_runs` — Workflow execution runs
- `workflow_step_runs` — Workflow step execution records

### Named Functions (FaaS)

- `function_definitions` — Function definition records
- `function_definition_versions` — Published function versions
- `function_runs` — Function invoke / try runs

### Events

- `data_events` — Durable business events
- `event_outbox` — Transactional outbox for event publishing
- `polling_state` — Per-provider polling cursor state

### Notifications

- `notify_channels` — Per-user notification channel configuration
- `notify_rules` — Notification gateway rules
- `notify_templates` — Notification templates
- `notification_records` — Notification delivery history

### Web Auth

- `web_accounts` — Web console accounts (password / TOTP)

### Agent / Chat Agent

- `agents` — Desktop agent records
- `agent_skills` / `agent_skill_files` — Agent skill registrations and files
- `agent_knowledge` — Knowledge documents
- `agent_memory_facts` — Memory facts
- `agent_plans` — Agent plans
- `agent_session_summaries` — Session summaries
- `agent_subagents` / `agent_subagent_tasks` — Subagent definitions and tasks
- `agent_todos` — Agent todo items
- `chat_sessions` / `chat_session_entries` — Chat session state and turns
- `chat_scheduled_tasks` / `chat_scheduled_task_runs` — Scheduled chat tasks
- `llm_usage_records` — LLM token usage

### Clips

- `clips` — Clip resources (public/private visibility)

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
- `file_uploads` — File upload records

### Analytics

- `behavior` — User behavior statistics
- `counters` / `counter_records` — Counter system

### Audit

- `audit_logs` — Audit log entries

## Database Schema Management

Ent auto-migration via `client.Schema.Create()` on startup. No manual SQL migrations.

Removing an Ent entity does **not** drop its PostgreSQL table. After upgrades that delete schemas, run the orphan-table `DROP` statements in [Database schema upgrades](../developer-guide/deployment.md#database-schema-upgrades).

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
