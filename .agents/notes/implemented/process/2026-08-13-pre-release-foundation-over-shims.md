# Agent Note: Foundation over compatibility shims (0.x)

Status: implemented

## Problem

Tagged 0.x releases exist, but there is no supported external Go-module or plugin ABI. Compatibility shims for old HTTP paths, config keys, and on-disk formats spread the blast radius of a wrong foundation: every later change pays for both shapes. Agents default to "don't break callers" even when the only callers are this repository.

## Decision

Until 1.0 or a stable public API is declared, prefer the correct foundation over compatibility shims. Rename or reshape freely and update every in-repo reference together. Config loaders reject legacy keys rather than dual-read. Do not add shims for old HTTP paths, config keys, or on-disk formats unless an Agent Note records a concrete consumer that cannot move in the same change.

**Recorded consumer contracts are not freely renamed.** Domain event names stay stable; that contract lives in [pkg/capability/AGENTS.md](../../../../pkg/capability/AGENTS.md). Pipeline and webhook subscribers cannot move in the same rename.

Root `AGENTS.md` carries a one-to-three-line standing order linking this note; rewrite that order at 1.0. This note owns the rationale. Product operators on tagged 0.x releases are not a reason to keep a dual code path; they move with the release notes.

## Alternatives considered

- **Treat 0.x tags as a stable public API.** Rejected: the tags ship the application, not a library contract. Dual-read and alias routes freeze mistakes in place.
- **Always keep one release of backward compatibility.** Rejected: without an identified external consumer of the old shape, the shim is owned complexity with no caller.

## Consequences

- Breaking in-repo renames are done in one change: code, tests, docs, and generated artifacts together.
- A real external consumer (a documented integration that cannot update in lockstep) is recorded in an Agent Note before a shim is added.
- Root `AGENTS.md` carries a one-to-three-line standing order linking this note.

## Verification

Config loaders reject legacy keys rather than dual-read ([docs/reference/config-reference.md](../../../../docs/reference/config-reference.md)). Domain event names stay stable in [pkg/capability/AGENTS.md](../../../../pkg/capability/AGENTS.md). A shim requires a new Agent Note that names the stuck consumer.
