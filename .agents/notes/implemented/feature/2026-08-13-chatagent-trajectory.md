# Agent Note: Chatagent trajectory view

Status: implemented

## Problem

The chat UI shows the user-visible transcript. It does not show what the model was injected with on a given turn (system body, project files, skills, memory, runtime), nor a Duration breakdown of prompt assemble vs model vs tools. Operators debugging a runaway or empty reply have to guess from bubbles. Reconstructing SYSTEM/CONTEXT from today's files would lie about historical turns.

## Decision

Trajectory is a second reading of the same session on `/service/web/agents/:id` (Chat | Trajectory). It is not a second store.

Each interactive run appends one `turn_trace` session-tree node after prompt assembly and before `Harness.Prompt`. The engine type is generic (`Sections []{Name,Content,Hash}` plus `AssembleMs`). chatagent fills section names `system_body`, `context_file`, `skills`, `memory`, and `runtime`. The node is ignored by `BuildContext`, cut-point selection, and compaction serialization, so it never enters the LLM or token ring.

`GET …/trajectory` (REST and Web) joins the branch with those nodes into labeled rows. The messages SSE emits `turn_trace` on the primary channel only (not `/events`). One `Service.Run` writes one snapshot; inner tool loops share it. Pipeline / ephemeral pipeline runs skip the node. Web, REST, platform DM, and scheduled runs persist it.

The first UI is a role log, a Duration gantt (Input = assemble, Model = thinking + remaining turn, Tools = tool durations), and a Preview/Raw inspector. Missing historical `turn_trace` nodes omit SYSTEM/CONTEXT rather than fabricating them.

## Alternatives considered

- **Chat-page view vs inspect-only vs both.** Inspect-only would hide the diagnostic from the primary session. Duplicating the UI on A-05 was extra surface. A drawer cannot hold gantt + inspector.
- **Project existing `HistoryMessage` only.** SYSTEM/CONTEXT are rebuilt at request time and are not in the message tree; showing current `AGENTS.md` as the past injection is a shim.
- **Parallel trajectory event table.** A second append log would desync from compaction and branch walks.
- **`EntryCustom` payload convention.** Product schema would be opaque to marshal tests and compaction guards.
- **Full `convertToLLM` dump.** Duplicates the session tree inside the snapshot.
- **Hash-only / reread files at display time.** File edits would rewrite history.
- **Turns/Calls modes, search, Source, virtualization, nested subagents.** Deferred; Duration + labeled log + Preview/Raw is the diagnostic core.
- **SSE stub plus a second GET.** Rejected for v1 so the live view has the injection immediately; frames may be tens of KB.

## Consequences

- Session trees grow by one JSON payload per user run (full section text; hashes reserved for later copy-on-change).
- Messages SSE `turn_trace` frames carry that full section text (tens of KB). v1 does not stub the frame and refetch.
- Export and A-05 inspect include `turn_trace` entries without a dedicated inspect UI.
- `h.Prompt` failure after append leaves an orphan snapshot; the next user message parents to it.
- Trajectory IA, `GET …/trajectory`, and the messages payload stay unchanged. Chat feed chrome is owned by [chatagent-chat-feed-style](2026-08-13-chatagent-chat-feed-style.md).

## Verification

- `pkg/agent/session`: `turn_trace` round-trips; `BuildContext` skips it.
- `pkg/agent/ctxmgr`: cut point and `GetContextUsage` ignore `turn_trace`.
- `internal/server/chatagent`: pipeline skips persist; interactive persist + SSE `turn_trace`; `GET …/trajectory` assembles rows; `IsObserverStreamEvent` excludes `turn_trace`.
- Web: Chat | Trajectory toggle; `?view=trajectory`; `public/js/chatagent-trajectory.js`.
- Checklist: [docs/agent/chatagent-feature-checklist.md](../../../docs/agent/chatagent-feature-checklist.md) R-03a / W-10 / O-13.
