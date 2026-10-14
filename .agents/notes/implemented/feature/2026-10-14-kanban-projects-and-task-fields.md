# Agent Note: Kanban projects and task fields

Status: implemented

## Problem

Prod CLI write paths (`kanban create` / `update`) crashed before any HTTP call because global `--debug/-d` and local `--description/-d` both claimed shorthand `-d`. Separately, Flowbot had no Kanboard project list/create surface, and create/update could not set due date, swimlane, or category — so boards that needed a dedicated project or those fields could not be bootstrapped from CLI.

## Decision

- Keep global `-d` for `--debug`. Drop the short option from kanban/trello description flags; use long `--description` / `--desc` only.
- Add capability ops `list_projects`, `create_project`, `list_swimlanes`, `list_categories`, wired through hub static routes and `pkg/client`, then CLI (`kanban project|swimlane|category`).
- Extend create/update task contracts with `date_due` (unix), `category_id`, and `swimlane_id`; CLI `--due` accepts `YYYY-MM-DD` (local midnight → unix). Expose `--swimlane` on move as well.
- Update fields use optional pointers so an explicit `0` clears due/category/swimlane; omitted fields are left unchanged. Invalid `project_id` query values return 400.
- Milestones stay as `[M0]`–`[M5]` tags; no milestone entity in this change.

## Alternatives considered

- Change global debug short flag away from `-d`: rejected — debug is the shared root flag and already documented across skills.
- Give description `-m` like Firefly: rejected — dropping the short option is clearer and avoids another collision class.
- Hub calling provider directly for projects (like tags): rejected — new surface follows the capability Invoke path used by tasks.

## Consequences

- Scripts that used `kanban create -d "..."` must switch to `--description`.
- Creating a project and assigning due/swimlane/category is a first-class CLI/API path; tag-only milestones remain the milestone convention.

## Verification

- `go test` covers CLI shorthand non-collision with root `--debug/-d`, provider RPC for projects/swimlanes/categories and update field passthrough, and capability registration ops count.
- `go tool task lint` passes on the changed packages.
- `go tool task skills` regenerates kanboard CLI skill docs without description `-d`.
