CREATE TABLE IF NOT EXISTS resource_links (
    id BIGSERIAL PRIMARY KEY,
    source_event_id TEXT NOT NULL,
    target_event_id TEXT NOT NULL,
    source_app TEXT NOT NULL DEFAULT '',
    target_app TEXT NOT NULL DEFAULT '',
    source_capability TEXT NOT NULL DEFAULT '',
    target_capability TEXT NOT NULL DEFAULT '',
    source_entity_id TEXT NOT NULL DEFAULT '',
    target_entity_id TEXT NOT NULL DEFAULT '',
    pipeline_run_id BIGINT,
    pipeline_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_event_id, target_event_id)
);
CREATE INDEX idx_rl_src_app_entity ON resource_links (source_app, source_entity_id);
CREATE INDEX idx_rl_tgt_app_entity ON resource_links (target_app, target_entity_id);
CREATE INDEX idx_rl_src_event ON resource_links (source_event_id);
CREATE INDEX idx_rl_tgt_event ON resource_links (target_event_id);
