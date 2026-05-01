CREATE TABLE IF NOT EXISTS `pipeline_runs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `pipeline_name` varchar(128) NOT NULL,
  `event_id` varchar(64) NOT NULL,
  `event_type` varchar(128) NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT '0',
  `error` text,
  `started_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_event_id` (`event_id`),
  KEY `idx_pipeline_name` (`pipeline_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
