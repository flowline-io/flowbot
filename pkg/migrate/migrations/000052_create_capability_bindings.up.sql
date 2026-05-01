CREATE TABLE IF NOT EXISTS `capability_bindings` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `capability` varchar(64) NOT NULL,
  `backend` varchar(64) NOT NULL,
  `app` varchar(64) NOT NULL,
  `healthy` tinyint NOT NULL DEFAULT '0',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_capability` (`capability`),
  KEY `idx_backend` (`backend`),
  KEY `idx_app` (`app`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
