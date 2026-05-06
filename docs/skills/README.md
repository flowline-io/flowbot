# Skills

Flowbot ships with AI assistant skills that teach Claude Code, opencode, and
other AI coding assistants how to use the `flowbot` CLI for daily tasks. Each
skill corresponds to a Flowbot capability and describes its full command tree,
flags, workflows, and troubleshooting tips.

Skills follow the SKILL.md convention. The AI assistant loads the skill's
frontmatter (name + description) at startup, and only pulls the full SKILL.md
body when the user's request matches the description.

## Available Skills

| Skill                   | Description                                              |
| ----------------------- | -------------------------------------------------------- |
| `homelab-bookmark`      | Create, search, and archive bookmarks / saved links      |
| `homelab-kanban`        | Manage kanban boards, tasks, and subtasks                |
| `homelab-reader`        | Subscribe to RSS/Atom feeds, read entries, mark status   |

Each skill file is in the corresponding subdirectory:

```
docs/skills/
├── homelab-bookmark/
│   └── SKILL.md
├── homelab-kanban/
│   └── SKILL.md
├── homelab-reader/
│   └── SKILL.md
└── README.md          (you are here)
```

## How Skills Work with AI Assistants

1. An AI assistant (Claude Code, opencode, etc.) reads `available_skills` at
   startup, which includes every skill's **name** and **description** from the
   SKILL.md YAML frontmatter.

2. When the user sends a message that matches a skill's description (e.g.
   "save this URL to my bookmarks"), the assistant loads the full SKILL.md body
   into its context.

3. The SKILL.md body contains:
   - Prerequisites (CLI login, server URL)
   - Global flags reference
   - Full command tree (nested operations with flags and arguments)
   - Common workflows (multi-step task recipes)
   - Troubleshooting guidance

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
ln -sf "$(pwd)/docs/skills/homelab-bookmark" .claude/skills/homelab-bookmark
ln -sf "$(pwd)/docs/skills/homelab-kanban"    .claude/skills/homelab-kanban
ln -sf "$(pwd)/docs/skills/homelab-reader"    .claude/skills/homelab-reader
```

For global availability:

```bash
# Symlink into the global Claude Code skills directory
mkdir -p ~/.claude/skills/
ln -sf "$(pwd)/docs/skills/homelab-bookmark" ~/.claude/skills/homelab-bookmark
ln -sf "$(pwd)/docs/skills/homelab-kanban"    ~/.claude/skills/homelab-kanban
ln -sf "$(pwd)/docs/skills/homelab-reader"    ~/.claude/skills/homelab-reader
```

## Generating Skills

Skills are generated from the live CLI command tree by the composer tool.
This ensures the skill always matches the actual CLI interface.

```bash
# Generate all SKILL.md files to docs/skills/
go tool task build:composer
./flowbot-composer skills --output ./docs/skills
```

When you add a new CLI command tree (kanban, bookmark, reader, etc.), register
it in `metaSpecs` in `cmd/composer/action/skills/skills.go` and re-run the
generator.

## Adding a New Skill

1. Implement the CLI command tree in `cmd/cli/command/` following existing
   conventions.

2. Register the capability in `metaSpecs` in
   `cmd/composer/action/skills/skills.go`:

   ```go
   {
       Name:        "homelab-myapp",
       Title:       "MyApp",
       CommandFn:   command.MyAppCommand,
       Description: "Manage MyApp resources via the Flowbot CLI.",
       Keywords:    "myapp, keywords, that trigger this skill",
       Workflows: []workflowSpec{
           {
               Title:       "Common task name",
               Description: "When the user wants to do X:",
               Steps: []workflowStep{
                   {Step: 1, Command: "flowbot myapp list"},
                   {Step: 2, Command: "flowbot myapp get <id>"},
               },
           },
       },
   },
   ```

3. Regenerate the skills:

   ```bash
   go tool task build:composer
   ./flowbot-composer skills --output ./docs/skills
   ```

   The new subdirectory and SKILL.md will appear under `docs/skills/`.

4. Add the skill directory to your AI assistant's skills path as shown above.

## Skill File Anatomy

Each `SKILL.md` follows this structure:

```markdown
---
name: homelab-bookmark
description: >
  What the skill does in one sentence.
  Make sure to use this skill whenever the user mentions <trigger keywords>.
---

# Flowbot Bookmark

## Prerequisites
## Global Flags Reference
## Common Output Options
## Operations (auto-generated from CLI tree)
## Common Workflows
## Troubleshooting
```

- **Frontmatter**: `name` (identifier) and `description` (trigger + summary).
- **Prerequisites**: CLI setup requirements.
- **Global Flags**: `--server-url`, `--profile`, `--debug`.
- **Operations**: Auto-generated command reference from the live `*cli.Command`
  tree. Flags are extracted via `RequiredFlag` and `DocGenerationFlag`
  interfaces.
- **Workflows**: Hand-written multi-step recipes for common tasks.
- **Troubleshooting**: Common errors and their solutions.

## References

- [Skill specification](../skill_spec.md) — Full format reference and
  conventions for SKILL.md files.
- [Composer skills code](../../cmd/composer/action/skills/skills.go) — The
  generator that produces these files.
- [CLI commands](../../cmd/cli/command/) — The CLI command tree implementations.
