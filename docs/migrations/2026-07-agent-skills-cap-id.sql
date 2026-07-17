-- Agent skill Cap ID rename migration
-- Run manually after upgrading Flowbot. Ent does not migrate row values.
-- Renames legacy homelab-* skill flag/name to hub.CapabilityType (provider ID).

BEGIN;

-- Prefer updating flag first (referenced by agent_skill_files.skill_flag), then name.
-- Skip rows that would collide with an already-migrated Cap ID skill.

UPDATE agent_skill_files SET skill_flag = 'karakeep'
WHERE skill_flag = 'homelab-bookmark'
  AND NOT EXISTS (SELECT 1 FROM agent_skills WHERE flag = 'karakeep');

UPDATE agent_skills
SET flag = 'karakeep',
    name = 'karakeep',
    base_dir = replace(base_dir, 'homelab-bookmark', 'karakeep')
WHERE flag = 'homelab-bookmark'
  AND NOT EXISTS (
    SELECT 1 FROM agent_skills AS existing
    WHERE existing.flag = 'karakeep' OR existing.name = 'karakeep'
  );

UPDATE agent_skill_files SET skill_flag = 'kanboard'
WHERE skill_flag = 'homelab-kanban'
  AND NOT EXISTS (SELECT 1 FROM agent_skills WHERE flag = 'kanboard');

UPDATE agent_skills
SET flag = 'kanboard',
    name = 'kanboard',
    base_dir = replace(base_dir, 'homelab-kanban', 'kanboard')
WHERE flag = 'homelab-kanban'
  AND NOT EXISTS (
    SELECT 1 FROM agent_skills AS existing
    WHERE existing.flag = 'kanboard' OR existing.name = 'kanboard'
  );

UPDATE agent_skill_files SET skill_flag = 'miniflux'
WHERE skill_flag = 'homelab-reader'
  AND NOT EXISTS (SELECT 1 FROM agent_skills WHERE flag = 'miniflux');

UPDATE agent_skills
SET flag = 'miniflux',
    name = 'miniflux',
    base_dir = replace(base_dir, 'homelab-reader', 'miniflux')
WHERE flag = 'homelab-reader'
  AND NOT EXISTS (
    SELECT 1 FROM agent_skills AS existing
    WHERE existing.flag = 'miniflux' OR existing.name = 'miniflux'
  );

COMMIT;

-- Manual checklist (not automated):
-- 1. Re-import / paste regenerated docs/skills/{cap}/SKILL.md content into agent_skills.content
--    (and references/cli.md as agent_skill_files) when using Cap ID directories.
-- 2. Pipeline / subagent skill allowlists: change homelab-bookmark -> karakeep, etc.
--    Runtime still resolves legacy names this release via chatagent.CanonicalSkillName.
-- 3. Symlinks under .claude/skills/homelab-* should point at docs/skills/{cap}/ instead.
-- 4. If both old and new rows exist, delete the leftover homelab-* row after verifying Cap ID content.
