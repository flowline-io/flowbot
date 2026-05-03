ALTER TABLE `pipeline_step_runs`
  ADD COLUMN `attempt` int NOT NULL DEFAULT 1,
  ADD COLUMN `retry_config` json DEFAULT NULL;
