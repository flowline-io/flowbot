# Agent Note: Chatagent thread header meta chips

Status: implemented

## Problem

The thread header stacked title, session id, model+thinking, and workspace as four similar text lines. Hierarchy was weak, the long session flag competed with the title, and model/thinking were concatenated into one English string that did not match the rest of the localized UI.

## Decision

Render session id, model, thinking level, and workspace as one wrapping row of muted `flowbot-chip` items with small icons. Title stays the only large heading. Session flag remains desktop-only and truncates inside the chip. Settings save updates the model and thinking chip text nodes, not a concatenated header line.

## Alternatives considered

- **Keep stacked paragraphs, tighten type.** Rejected: the id line still dominated, and related facts stayed visually equal.
- **Labeled key/value columns.** Rejected: heavier than an ops-console chip row and fights the existing `.flowbot-chip` pattern.

## Consequences

- Header copy no longer uses `chatagent.workspace` as a visible prefix; chip `title` attributes reuse `chatagent.settings.*_aria`.
- `chatAgentSessionSettingsLabel` is gone; thinking chips use `chatAgentThinkingI18nKey` plus catalog keys.

## Verification

- `TestChatAgentThreadHeaderMobile` asserts the meta row, chips, desktop-only session id class, and workspace value.
- `TestChatAgentThinkingI18nKey` maps levels to `chatagent.settings.thinking.*`.
