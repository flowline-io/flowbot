ALTER TABLE `flow_queue_jobs`
    ADD COLUMN `execution_id` VARCHAR(64) NOT NULL DEFAULT '' AFTER `flow_id`,
    ADD INDEX `idx_flow_queue_execution_id` (`execution_id`);
