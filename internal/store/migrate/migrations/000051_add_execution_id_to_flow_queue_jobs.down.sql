ALTER TABLE `flow_queue_jobs`
    DROP INDEX `idx_flow_queue_execution_id`,
    DROP COLUMN `execution_id`;
