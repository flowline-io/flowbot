# Agent Note: One home per fact

Status: implemented

## Problem

The same rule was easy to copy into root `AGENTS.md`, architecture overviews, and README. Copies drift. Agents treat every copy as authority and pad each file until none of them is the map.

## Decision

Each fact has one home: the documentation tier whose job it is. Other files link there. Root `AGENTS.md` holds standing orders of one to three lines. Architecture docs hold the current system map. pkg vs internal lives in [pkg-boundaries.md](../../../../docs/architecture/pkg-boundaries.md). README holds the product pitch and how to run it. Rationale lives in Agent Notes. The tier table and writing rules live in [docs/AGENTS.md](../../../../docs/AGENTS.md). Subtree `AGENTS.md` files keep only tree-specific orders and link root for the rest.

## Alternatives considered

- **Repeat the full rule wherever an agent might miss the link.** Rejected: the copies become the drift. A short standing order plus a link is the enforcement surface.
- **A generated index of every rule.** Rejected: the tree position is the index; another catalog would itself need a home and a freshness gate.

## Consequences

- Adding a rule means choosing its home first, then a one-line standing order if agents need it every session.
- Grep a distinctive phrase before restating a rule. Keep one home and link the rest.

## Verification

The tier table lives only in [docs/AGENTS.md](../../../../docs/AGENTS.md). Root `AGENTS.md` standing orders are one to three lines with a link. pkg vs internal is only in [pkg-boundaries.md](../../../../docs/architecture/pkg-boundaries.md). Architecture [README](../../../../docs/architecture/README.md) links standing orders instead of restating them.
