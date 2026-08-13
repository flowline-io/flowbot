# Agent Notes

An **Agent Note** is the RFC for a decision that affects this codebase — the *why*, *what we gave up*, and *how we will know it worked*. Code, tests, and docs carry the resulting contract; they do not carry the trade-off.

## Layout and naming

Path: `{lifecycle}/{class}/yyyy-mm-dd-topic-title.md`. The date is when the topic was first proposed.

Lifecycle (the file moves as status changes):

- **`proposed/`** — not yet built, or only partly.
- **`implemented/`** — the decision shipped. Keep the file current with what actually shipped (paths, names, defaults — not a rewrite of the decision). See [implemented/AGENTS.md](implemented/AGENTS.md).
- **`rejected/`** — considered and declined. Keep it only while the rationale still prevents a tempting mistake; otherwise delete it.

Class (closed set):

| Class | Covers |
| --- | --- |
| `feature` | A new user-facing capability |
| `bug-fix` | A defect or a gap a failure surfaced |
| `simplification` | Removes code, behavior, or surface without adding a capability |
| `architecture` | How shipped source is structured |
| `process` | Tooling, policy, or workflow around the code |
| `testing` | Test infrastructure and strategy |

Cross-references use relative Markdown links, never bare filenames or note numbers.

## When to write one

Every non-trivial change adds or updates at least one Agent Note in the same PR. A change is non-trivial when it alters behavior, architecture, a contract shared across packages, process or tooling, testing strategy, an on-disk / wire / configuration format, or another decision a maintainer may reasonably revisit.

Updating the note that already owns the decision satisfies the rule. Only a purely mechanical or local edit with no change to behavior, contracts, structure, process, or rationale is exempt.

Do not edit an Agent Note into a *different* decision: supersede it with a new note and cross-link both.

## File format

First lines:

```markdown
# Agent Note: <title>

Status: <status>
```

`Status:` is `proposed`, `implemented`, or `rejected — <why, in one line>`, and must match the lifecycle folder.

Every note opens with `## Problem` (motivation that stands without the solution) and includes `## Alternatives considered` (each genuine alternative and why it lost). Invented alternatives are not a substitute for recorded ones.

### `proposed/`

```markdown
## Problem
## Proposal
## Alternatives considered
## Acceptance criteria
## Risks
```

`## Proposal` may use future tense. `## Acceptance criteria` says what observable state means done.

### `implemented/`

```markdown
## Problem
## Decision
## Alternatives considered
## Consequences
## Verification
```

`## Decision` describes shipped reality in the present tense. `## Verification` is present-tense evidence that the decision holds (tests, gates, owning docs) — not proposal-era `## Acceptance criteria`. Proposal-era headings (`## Proposal`, `## Plan`, `## Migration plan`, `## Acceptance criteria`) do not belong here.

### `rejected/`

Keeps proposal-time sections. The verdict lives on the `Status:` line.

## Archiving

Archive an implemented note when the shipped decision is complete and its rationale is unlikely to guide future work. Move it to `archived/{class}/yyyy-mm-dd-topic-title.md`, keep `Status: implemented`, and insert `Archived: YYYY-MM-DD` immediately below that line. Those are the only permitted content changes during archival.

Once sealed, an archived note is frozen: do not edit, reformat, update, move, or delete it, and do not treat it as authority for current behavior. Active prose may link into an archived note when it intentionally cites history. See [archived/AGENTS.md](archived/AGENTS.md).
