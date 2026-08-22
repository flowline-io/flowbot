# Agent Note: Context Usage Skills collapsed by default

Status: implemented
Archived: 2026-08-22

## Problem

The Context Usage popover always listed every injected skill under Skills. With many enabled skills the breakdown is taller than the other categories and the per-skill names are usually noise unless the operator is debugging token spend.

## Decision

When the report has at least one skill, Skills is a closed `<details>` row. Opening the popover starts collapsed. A live refresh while the popover stays open keeps the operator's open/closed state. Reopening the popover starts collapsed again.

## Alternatives considered

- **Always expanded.** Status quo; the list dominates the popover.
- **Drop per-skill rows.** Loses the token attribution the category exists for.
- **Remember open state across popover close.** Rejected: "default collapsed" should hold each time the operator opens the ring.

## Consequences

- Skill names are one click away; the Skills token total stays visible on the summary row.
- `public/js/chatagent-context.js` owns the disclosure; the popover template stays a shell.

## Verification

- W-05: Skills list collapsed until opened ([chatagent-feature-checklist](../../../../docs/agent/chatagent-feature-checklist.md)).
- Web: open the context ring on `/service/web/agents/:id`; Skills shows a caret and no skill names until expanded.
