# Documentation standard

This file is the home for how contributor docs are placed and written. Standing orders stay in root [AGENTS.md](../AGENTS.md) (one to three lines each). Decision rationale lives in [Agent Notes](../.agents/notes/README.md).

## One home per fact

Each fact has one home: the tier whose job it is. Elsewhere, link there. Do not write the same rule in `AGENTS.md`, architecture docs, and README.

| Tier | Job | Does not belong there |
| --- | --- | --- |
| Root `AGENTS.md` | Standing orders an agent needs every session, one to three lines each, linking the home | Stories, worked examples, procedures, anything restated from a linked home |
| Subtree `AGENTS.md` | Orders specific to that package tree | Repo-wide rules the root file already carries |
| [architecture/](architecture/README.md) | System map: layers, data flows, diagrams | Decision rationale (→ Agent Notes), test mechanics (→ testing docs), restated standing orders |
| [pkg-boundaries.md](architecture/pkg-boundaries.md) | pkg vs internal import and API surface | The same bullets copied into subtree `AGENTS.md` |
| [Agent Notes](../.agents/notes/README.md) | Decision records: why, what was given up, required verification | Format restated from [notes README](../.agents/notes/README.md); current-behavior standing orders |
| Package / module `example/` | The runnable reference for that layer | Policy restated from root `AGENTS.md` |
| [testing/](testing/README.md) | When to test, which layer, assembled-example rule | Architecture diagrams, dependency policy |
| [developer-guide/](developer-guide/README.md) | How to build, deploy, and operate | Standing orders and decision rationale |
| Product README | What the product is and how to run it | Contributor standing orders |

Placement: rationale → Agent Notes; procedures → developer-guide or cookbooks; standing orders → root `AGENTS.md` with a link to the home; architecture → the current map only.

## Writing rules

- Document current state, not change history. Avoid "previously / now / no longer" in durable prose; name the live mechanism. Put change stories in commits, PRs, Agent Notes.
- Comments and godoc state contracts, not reasoning transcripts. Do not narrate control flow, preserve review history, or restate code.
- Before repeating a rule, grep a distinctive phrase. Keep one home and link the rest.
