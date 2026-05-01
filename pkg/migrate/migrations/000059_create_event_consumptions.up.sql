CREATE TABLE IF NOT EXISTS `event_consumptions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `consumer_name` varchar(128) NOT NULL,
  `event_id` varchar(64) NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_consumer_event` (`consumer_name`, `event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
