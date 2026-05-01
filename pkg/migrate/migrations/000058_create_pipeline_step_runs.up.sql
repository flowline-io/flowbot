CREATE TABLE IF NOT EXISTS `pipeline_step_runs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `pipeline_run_id` bigint NOT NULL,
  `step_name` varchar(128) NOT NULL,
  `capability` varchar(64) NOT NULL DEFAULT '',
  `operation` varchar(64) NOT NULL DEFAULT '',
  `params` json DEFAULT NULL,
  `result` json DEFAULT NULL,
  `status` tinyint NOT NULL DEFAULT '0',
  `error` text,
  `started_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_pipeline_run_id` (`pipeline_run_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
