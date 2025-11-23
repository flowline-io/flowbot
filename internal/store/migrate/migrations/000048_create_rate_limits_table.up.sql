CREATE TABLE IF NOT EXISTS `rate_limits`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `flow_id`     BIGINT(20) UNSIGNED NULL,
    `node_id`     VARCHAR(64)         NULL COLLATE 'utf8mb4_unicode_ci',
    `limit_type` VARCHAR(32)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `limit_value` INT(11)             NOT NULL DEFAULT '0',
    `window_size` INT(11)             NOT NULL DEFAULT '60',
    `window_unit` VARCHAR(8)          NOT NULL DEFAULT 'second' COLLATE 'utf8mb4_unicode_ci',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `flow_id` (`flow_id`) USING BTREE,
    INDEX `node_id` (`node_id`) USING BTREE,
    INDEX `limit_type` (`limit_type`) USING BTREE,
    CONSTRAINT `fk_rate_limits_flow` FOREIGN KEY (`flow_id`) REFERENCES `flows` (`id`) ON DELETE CASCADE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

