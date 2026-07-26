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
internal/store/store.go         # LifeStore facade (no life_store.go)
internal/modules/web/           # life_*_webservice.go, SetLifeService, User nav
pkg/views/pages|partials/       # life_*.templ
```

## UI entry

Signed-in **User** nav (session badge dropdown): Life, Character, Quests, Inventory — under `/service/web/life*`.

## Reference docs

Database reference / `task webdoc` links are updated after schema ships; see [Architecture](./architecture.md) for the table list until then.
