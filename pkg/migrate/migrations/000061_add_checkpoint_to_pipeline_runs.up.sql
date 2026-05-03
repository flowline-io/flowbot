ALTER TABLE `pipeline_runs`
  ADD COLUMN `checkpoint_data` json DEFAULT NULL,
  ADD COLUMN `last_heartbeat` datetime DEFAULT NULL;
