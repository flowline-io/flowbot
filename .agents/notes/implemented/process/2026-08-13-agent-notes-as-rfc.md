# Agent Note: Agent Notes are RFCs

Status: implemented

## Problem

Non-trivial decisions were landing as code plus a short PR description. The problem, the alternatives given up, and the verification that would prove the decision were not recorded in a durable place. Agents and later maintainers re-litigated the same trade-offs, or treated outdated comments as current authority.

## Decision

An Agent Note is the RFC for a non-trivial change. The same PR that ships the change adds or updates a note that states the problem, the alternatives considered, and the verification required. After the decision ships, the note describes reality in the present tense and stays current with paths, names, and defaults. When the rationale no longer guides future work, the note is archived and frozen — never edited and never treated as current authority.

The format, when-to-write rule, and archive policy live in [.agents/notes/README.md](../../README.md). Root `AGENTS.md` carries the one-to-three-line standing order.

## Alternatives considered

- **PR description only.** Rejected: PRs are not the lookup home, and they disappear from the working tree.
- **Inline comments that retell the decision.** Rejected: comments state contracts, not RFCs; they rot next to the code they describe.
- **A single living design doc that is rewritten in place.** Rejected: it mixes current map with history and invites silent reversal of a decision.

## Consequences

- Mechanical and local edits stay exempt; everything else pays the note.
- Implemented notes are maintenance: stale paths are updated in the same change; a different decision is a new note.
- Archived notes stop being a source of standing orders.

## Verification

Format and when-to-write live in [.agents/notes/README.md](../../README.md). Implemented notes in `.agents/notes/implemented/` use present tense, include `## Verification`, and omit proposal-era headings. Archived notes under `.agents/notes/archived/` are not edited.
