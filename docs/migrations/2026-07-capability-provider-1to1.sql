-- Capability 1:1 provider rename migration
-- Run manually after upgrading Flowbot. Ent does not migrate row values.

BEGIN;

-- capability_bindings: map legacy domain capability names to provider IDs
UPDATE capability_bindings SET capability = 'karakeep' WHERE capability = 'bookmark';
UPDATE capability_bindings SET capability = 'miniflux' WHERE capability = 'reader';
UPDATE capability_bindings SET capability = 'kanboard' WHERE capability = 'kanban';
UPDATE capability_bindings SET capability = 'trilium'  WHERE capability = 'note';
UPDATE capability_bindings SET capability = 'memos'    WHERE capability = 'memo';
UPDATE capability_bindings SET capability = 'gitea'    WHERE capability = 'forge';

-- data_events: capability column only
UPDATE data_events SET capability = 'karakeep' WHERE capability = 'bookmark';
UPDATE data_events SET capability = 'miniflux' WHERE capability = 'reader';
UPDATE data_events SET capability = 'kanboard' WHERE capability = 'kanban';
UPDATE data_events SET capability = 'trilium'  WHERE capability = 'note';
UPDATE data_events SET capability = 'memos'    WHERE capability = 'memo';
UPDATE data_events SET capability = 'gitea'    WHERE capability = 'forge';

-- Drop redundant backend columns (CapType == provider ID)
ALTER TABLE capability_bindings DROP COLUMN IF EXISTS backend;
ALTER TABLE data_events DROP COLUMN IF EXISTS backend;

COMMIT;

-- Manual checklist (not automated):
-- 1. Pipeline / workflow definitions: change capability: bookmark -> karakeep, etc.
-- 2. Auth token scopes: service:bookmark:* still authorize this release via HasScope aliases;
--    prefer re-issuing tokens with service:karakeep:* when convenient.
-- 3. Homelab compose labels: flowbot.capability=karakeep (legacy bookmark still maps this release)
-- 4. EventType strings (bookmark.created) are UNCHANGED — do not rename notify rules by event
-- 5. Prometheus metrics renamed ability_* -> capability_*; capability label values are provider IDs
