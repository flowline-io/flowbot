# Skills

Flowbot ships with AI assistant skills that teach Claude Code, opencode, and
other AI coding assistants how to use the `flowbot` CLI for daily tasks. Each
skill corresponds to a Flowbot capability: the skill **name equals the
capability ID** (`hub.CapabilityType` / provider ID). The skill body describes
the CLI command tree for that capability (CLI domain names may differ from the
capability ID, e.g. `karakeep` → `flowbot bookmark`).

Skills follow the SKILL.md convention. The AI assistant loads the skill's
frontmatter (name + description) at startup, and only pulls the full SKILL.md
body when the user's request matches the description. Additional files in the
skill directory (for example `reference.md` or `scripts/run.sh`) can be loaded
via `read_skill` with the `path` argument.

## Available Skills

| Skill (Cap ID) | CLI root   | Description                                            |
| -------------- | ---------- | ------------------------------------------------------ |
| `karakeep`     | `bookmark` | Create, search, and archive bookmarks / saved links    |
| `kanboard`     | `kanban`   | Manage kanban boards, tasks, and subtasks              |
| `miniflux`     | `reader`   | Subscribe to RSS/Atom feeds, read entries, mark status |
| `memos`        | `memo`     | Create, list, update, and delete memos                 |
| `trilium`      | `trilium`  | Create, list, search, update, and delete trilium notes |
| `gitea`        | `forge`    | Inspect forge repos, issues, diffs, and files          |
| `github`       | `github`   | Inspect GitHub repos, issues, notifications, releases  |

Each skill file is in the corresponding subdirectory:

```
docs/skills/
├── karakeep/
│   ├── SKILL.md
│   └── references/cli.md
├── kanboard/
│   ├── SKILL.md
│   └── references/cli.md
├── miniflux/
│   ├── SKILL.md
│   └── references/cli.md
├── memos/
│   ├── SKILL.md
│   └── references/cli.md
├── trilium/
│   ├── SKILL.md
│   └── references/cli.md
├── gitea/
│   ├── SKILL.md
│   └── references/cli.md
├── github/
│   ├── SKILL.md
│   └── references/cli.md
└── README.md          (you are here)
```

Capabilities without a CLI tree (`notify`, `agent`, `example`) are
not generated as skills.

## How Skills Work with AI Assistants

1. An AI assistant (Claude Code, opencode, etc.) reads `available_skills` at
   startup, which includes every skill's **name** and **description** from the
   SKILL.md YAML frontmatter.

2. When the user sends a message that matches a skill's description (e.g.
   "save this URL to my bookmarks"), the assistant loads the full SKILL.md body
   into its context.

3. The SKILL.md body contains:
   - Setup (login, server URL, output format)
   - Common workflows (multi-step recipes)
   - Troubleshooting
   - A link to `references/cli.md` for the full command tree (loaded on demand)

4. The assistant follows the instructions in the skill to compose the correct
   `flowbot` CLI commands and handle errors.

### Enabling Skills in Your Environment

Skills files are already generated in `docs/skills/`. To use them with your AI
assistant, configure the tool to scan this directory.

**opencode**: Add a `--skill-dir` or equivalent path pointing to this directory.

**Claude Code**: Place the skill directories under your global skills path.
For example, to make them available project-wide:

```bash
# Symlink Flowbot skills into your project's .claude/skills/
mkdir -p .claude/skills/
ln -sf "$(pwd)/docs/skills/karakeep" .claude/skills/karakeep
ln -sf "$(pwd)/docs/skills/kanboard" .claude/skills/kanboard
ln -sf "$(pwd)/docs/skills/miniflux" .claude/skills/miniflux
ln -sf "$(pwd)/docs/skills/memos"    .claude/skills/memos
ln -sf "$(pwd)/docs/skills/trilium"  .claude/skills/trilium
ln -sf "$(pwd)/docs/skills/gitea"    .claude/skills/gitea
ln -sf "$(pwd)/docs/skills/github"   .claude/skills/github
```

For global availability:

```bash
# Symlink into the global Claude Code skills directory
mkdir -p ~/.claude/skills/
ln -sf "$(pwd)/docs/skills/karakeep" ~/.claude/skills/karakeep
ln -sf "$(pwd)/docs/skills/kanboard" ~/.claude/skills/kanboard
ln -sf "$(pwd)/docs/skills/miniflux" ~/.claude/skills/miniflux
ln -sf "$(pwd)/docs/skills/memos"    ~/.claude/skills/memos
ln -sf "$(pwd)/docs/skills/trilium"  ~/.claude/skills/trilium
ln -sf "$(pwd)/docs/skills/gitea"    ~/.claude/skills/gitea
ln -sf "$(pwd)/docs/skills/github"   ~/.claude/skills/github
```

## Generating Skills

Skills are generated from the live CLI command tree by the composer tool.
This ensures the skill always matches the actual CLI interface.

```bash
# Generate all SKILL.md files to docs/skills/
go tool task skills
```

When you add a new CLI command tree for a capability, register it in
`metaSpecs` in `cmd/composer/action/skills/skills.go` (Name = capability ID)
and re-run the generator.

### Import into the database

The server binary embeds `docs/skills` (package `skills`) and **upserts** them
into `agent_skills` / `agent_skill_files` on every startup (after DB migrate in
`cmd` → `server.newDatabaseAdapter`). `source` is set to `bundled`. Existing
`enabled` / `disable_model_invocation` values are preserved on update.

Flow:

1. `go tool task skills` — regenerate markdown under `docs/skills/`
2. Rebuild/restart the server (`go tool task run`) — auto-import into Postgres
3. Chat agent loads skills from DB via `read_skill` / system prompt

No manual Web UI paste or composer `--sync-db` step is required for bundled skills.

## Adding a New Skill

1. Implement the CLI command tree in `cmd/cli/command/` following existing
   conventions.

2. Register the capability in `metaSpecs` in
   `cmd/composer/action/skills/skills.go`. **Name must equal the capability ID**
   from `pkg/hub` (e.g. `hub.CapKarakeep`), not the CLI domain name.
   (`hub.CapExample` is shown only as a pattern — example has no CLI skill.)

   ```go
   {
       Name:        string(hub.CapKarakeep),
       Title:       "Karakeep",
       CommandFn:   command.BookmarkCommand,
       Description: "Create, list, search, archive, and delete bookmarks via flowbot bookmark.",
       Keywords:    "bookmarks, karakeep, saved URLs",
       Workflows: []workflowSpec{
           {
               Title:       "Common task name",
               Description: "When the user wants to do X:",
               Steps: []workflowStep{
                   {Step: 1, Command: "flowbot bookmark list"},
                   {Step: 2, Command: "flowbot bookmark get <id>"},
               },
           },
       },
   },
   ```

3. Regenerate the skills:

   ```bash
   go tool task skills
   ```

   The new subdirectory and SKILL.md will appear under `docs/skills/`.

4. Add the skill directory to your AI assistant's skills path as shown above.

## Skill File Anatomy

Each skill directory follows Agent Skills progressive disclosure:

```
{capability-id}/
├── SKILL.md              # frontmatter + setup + workflows (lean)
└── references/
    └── cli.md            # full CLI command/flag reference (on demand)
```

```markdown
---
name: karakeep
description: >-
  Create, list, search, archive, and delete bookmarks via flowbot bookmark.
  Use when the user mentions bookmarks, karakeep, ...
compatibility: Requires flowbot CLI, network access to a Flowbot server
metadata:
  capability: karakeep
  cli_root: bookmark
---

# Karakeep

## Setup

## Workflows

## Troubleshooting
```

- **Frontmatter**: `name` equals capability ID and directory name; `description`
  is WHAT + WHEN (≤1024 chars); `metadata.capability` mirrors the Cap ID.
- **SKILL.md body**: Imperative setup, workflows, and troubleshooting only.
- **references/cli.md**: Auto-generated command tree from cobra (loaded only when needed).
- **Workflows**: Hand-written multi-step recipes for common tasks.

## References

- [Composer skills code](../../cmd/composer/action/skills/skills.go) — The
  generator that produces these files.
- [Capability types](../../pkg/hub/capability.go) — Canonical capability IDs.
- [CLI commands](../../cmd/cli/command/) — The CLI command tree implementations.
- [Skill Cap ID migration](../migrations/2026-07-agent-skills-cap-id.sql) —
  Rename `homelab-*` agent_skills rows to Cap IDs after upgrade.

## Migrating from `homelab-*` names

Skill **name** and directory now equal the capability ID. If you previously
stored skills as `homelab-bookmark` / `homelab-kanban` / `homelab-reader`:

1. Run [`docs/migrations/2026-07-agent-skills-cap-id.sql`](../migrations/2026-07-agent-skills-cap-id.sql).
2. Update pipeline / subagent allowlists to Cap IDs (`karakeep`, `kanboard`,
   `miniflux`). Runtime still resolves the old names this release.
3. Point Claude/opencode skill symlinks at `docs/skills/{cap}/`.
