CREATE TABLE IF NOT EXISTS `workflow_runs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `workflow_name` varchar(128) NOT NULL,
  `workflow_file` varchar(512) NOT NULL,
  `status` tinyint NOT NULL DEFAULT '0',
  `trigger_type` varchar(32) DEFAULT '',
  `trigger_info` json DEFAULT NULL,
  `input_params` json DEFAULT NULL,
  `checkpoint_data` json DEFAULT NULL,
  `last_heartbeat` datetime DEFAULT NULL,
  `error` text,
  `started_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_workflow_name` (`workflow_name`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
