CREATE TABLE IF NOT EXISTS `key_result_values`
(
    `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
    `key_result_id` bigint                   DEFAULT NULL,
    `value`         int             NOT NULL,
    `memo`          VARCHAR(1000)   NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `created_at`    datetime        NOT NULL,
    `updated_at`    datetime        NOT NULL,
    PRIMARY KEY (`id`),
    KEY `key_result_id` (`key_result_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;