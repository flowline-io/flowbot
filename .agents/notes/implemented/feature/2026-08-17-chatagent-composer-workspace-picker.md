# Agent Note: Chatagent composer workspace picker

Status: implemented

## Problem

Chat agent file and shell tools always used the single `chat_agent.workspace` root. Homelab layouts typically keep several project directories under that root, so a new session had no way to lock tools, context files, and permission bounds onto one project.

## Decision

`chat_agent.workspace` remains the configured sandbox parent. The agents home composer lists that root plus its first-level non-hidden subdirectories. The choice is sent on `POST /service/web/agents` (and REST `POST /chatagent/sessions`) as a relative path: empty string for the config root, a single segment such as `foo` for a subdirectory.

The relative path is stored on `chat_sessions.workspace` at create time and is immutable afterward. `PUT` settings with a `workspace` field returns 400. Thread UI shows the locked directory as read-only text.

At run time, `WorkspaceForSession` joins the config root with the stored relative path. That absolute directory is the effective coding workspace (system prompt, tools, progress, sensors, resource reads) and is injected as `ChatHookDeps.WorkspaceRoot` so permission external-path checks use the session root, not the yaml root. A missing directory fails the run; history remains readable. Ephemeral and scheduled sessions keep an empty relative path (config root) and do not copy a source session workspace.

Composer remembers the last pick in localStorage and restores it when the option still exists.

Invalid or missing client picks return 400 (`types.ErrInvalidArgument`) and do not leave an active session. Validate the relative path before `CreateSession`; if later persist or settings fail, close the row. A missing subdirectory after create still serves history; Run / harness / context usage fail without falling back to the config root. Config root itself missing is a server/config error, not a client 400.

## Alternatives considered

- **Multiple configured roots.** Would change yaml schema and validation; a single parent with first-level children matches current homelab layout.
- **Free-form path under the root, or recursive picker.** Heavier UI and easier mis-selection; first-level is enough.
- **Change workspace mid-session.** Prompt cache, tool paths, and permission history would desync.
- **Keep config root as the permission bound while using the subdir only as cwd.** The picker would not mean "this project."
- **Dedicated list API.** Directory lists change rarely; SSR matches selectable models.

## Consequences

- Scheduled tasks spawned from a project session still run against the config root until a later change copies the relative path onto ephemeral sessions.
- `AgentInfo.Workspace` stays the global config root.
- Harness pool keys already include `workspace.Root`, so sessions in different subdirectories do not share a harness.

## Verification

- `internal/server/chatagent/workspace_test.go`: normalize, list (skip dotdirs), missing resolve is `ErrInvalidArgument`, config-root missing is not, create persist, sibling-path external.
- REST and Web create persist `workspace`; nested / missing dir is 400 with no leftover active session; PUT settings with `workspace` is 400.
- Agents page and REST message list remain readable after the session subdirectory is deleted; harness create fails.
- Agents page composer includes `data-testid="chatagent-composer-workspace"`.
