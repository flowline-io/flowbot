CREATE TABLE IF NOT EXISTS `counter_records`
(
    `counter_id` bigint unsigned NOT NULL DEFAULT (0),
    `digit`      int             NOT NULL,
    `created_at` datetime        NOT NULL,
    PRIMARY KEY (`counter_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;