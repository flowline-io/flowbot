CREATE TABLE IF NOT EXISTS `workflow_step_runs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `workflow_run_id` bigint NOT NULL,
  `step_id` varchar(128) NOT NULL,
  `step_name` varchar(256) DEFAULT '',
  `action` varchar(512) NOT NULL,
  `action_type` varchar(32) NOT NULL,
  `params` json DEFAULT NULL,
  `result` json DEFAULT NULL,
  `attempt` int NOT NULL DEFAULT '1',
  `status` tinyint NOT NULL DEFAULT '0',
  `error` text,
  `started_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_workflow_run_id` (`workflow_run_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
