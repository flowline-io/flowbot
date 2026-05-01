CREATE TABLE IF NOT EXISTS `data_events` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `event_id` varchar(64) NOT NULL,
  `event_type` varchar(128) NOT NULL,
  `source` varchar(64) NOT NULL DEFAULT '',
  `capability` varchar(64) NOT NULL DEFAULT '',
  `operation` varchar(64) NOT NULL DEFAULT '',
  `backend` varchar(64) NOT NULL DEFAULT '',
  `app` varchar(64) NOT NULL DEFAULT '',
  `entity_id` varchar(128) NOT NULL DEFAULT '',
  `idempotency_key` varchar(128) NOT NULL DEFAULT '',
  `uid` varchar(64) NOT NULL DEFAULT '',
  `topic` varchar(64) NOT NULL DEFAULT '',
  `data` json DEFAULT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_event_id` (`event_id`),
  KEY `idx_event_type` (`event_type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `event_outbox` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `event_id` varchar(64) NOT NULL,
  `payload` json DEFAULT NULL,
  `published` tinyint NOT NULL DEFAULT '0',
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_event_id` (`event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pipeline_definitions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `description` varchar(512) DEFAULT '',
  `enabled` tinyint NOT NULL DEFAULT '1',
  `trigger` json DEFAULT NULL,
  `steps` json DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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

CREATE TABLE IF NOT EXISTS `event_consumptions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `consumer_name` varchar(128) NOT NULL,
  `event_id` varchar(64) NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_consumer_event` (`consumer_name`, `event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
