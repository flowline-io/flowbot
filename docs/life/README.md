# Life

Life is Flowbot's solo gamified productivity surface: goals and quests, cascading stats, equipment loot, and an AI dungeon-master style evaluator.

It is distinct from:

| Name | Location | Purpose |
| ---- | -------- | ------- |
| **Life (this doc)** | `internal/modules/life`, `pkg/life`, `pkg/capability/life` | Solo RPG productivity domain |
| **Product PRD** | [`templ/prd.md`](../../templ/prd.md) | Game design / entity narrative (Chinese) |
| **Platform User** | `users` table / web session | Account identity — not game stats |
| **Ops console** | `/service/web/*` Agent/Automate/Integrate/System | Homelab operations UI |

## Documentation

| Document | Description |
| -------- | ----------- |
| [Architecture](./architecture.md) | Package map, schema, APIs, flows, boundaries |

## Target source layout

```text
internal/modules/life/          # Service, Bootstrap, outbox consumer, config
internal/modules/life/seed/     # embed JSON catalogs
pkg/life/                       # cascade, loot, buffs (no I/O)
pkg/capability/life/            # EvaluateQuest, GenerateInstanceLore
internal/store/ent/schema/      # life_*.go
internal/store/life.go         # LifeStore facade (package store; not a separate *_store.go)
internal/modules/web/           # life_*_webservice.go, SetLifeService, User nav
pkg/views/pages|partials/       # life_*.templ
```

## UI entry

Signed-in **User** nav (session badge): opens Life. Module sidebar: Life, Stats, Quests, Skills, Inventory, Achievements, Rewards, Character — under `/service/web/life*`.

## Reference docs

See [Architecture](./architecture.md) for the table list and HTTP routes. Catalog tables are also listed under [Database reference](../reference/database-reference.md).
