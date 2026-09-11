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

### Gateway

- `gateway_jobs` — Local CLI gateway jobs
- `gateway_workers` — Gateway worker registrations

### Clips

- `clips` — Clip resources (public/private visibility)

### Life (solo RPG productivity)

- `life_profiles` — Operator profile (level, exp, gold, class, pity)
- `life_characteristics` — Cascading stats (INT/PHY/WIL/CHA/CRE/FIN/WRI/FOC)
- `life_skills` — Skills under a characteristic
- `life_goals` — PARA goals
- `life_quests` — One-Time / Daily / Boss quests
- `life_ai_contexts` — DM personality + mood (1:1 profile)
- `life_equipments` — Equipment catalog templates (seeded)
- `life_inventories` — Owned equipment instances + lore overrides
- `life_equipped_slots` — Worn inventory ids per slot
- `life_loot_tables` — Drop tier → chance + item pool
- `life_action_logs` — Completion / dice / drop audit
- `life_rewards` — Player-defined real-life rewards (gold sink)
- `life_reward_redemptions` — Reward redeem audit (name/price snapshots)
- `life_achievements` / `life_achievement_progress` / `life_achievement_unlocks` — Achievement catalog and progress
- `life_habit_checkins` — Habit check-ins
- `life_action_specs` / `life_action_occurrences` / `life_action_dependencies` — Action graph
- `life_adjudications` — Adjudication records
- `life_evidence` — Evidence attachments
- `life_plan_nodes` — Plan tree nodes

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
