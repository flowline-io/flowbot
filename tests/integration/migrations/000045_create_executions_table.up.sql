CREATE TABLE IF NOT EXISTS `executions`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `flow_id`     BIGINT(20) UNSIGNED NOT NULL,
    `execution_id` VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `trigger_type` VARCHAR(32)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `trigger_id`  VARCHAR(64)          NULL COLLATE 'utf8mb4_unicode_ci',
    `state`       TINYINT(3)           NOT NULL DEFAULT '0',
    `payload`     JSON                 NULL,
    `variables`   JSON                 NULL,
    `result`      JSON                 NULL,
    `error`       TEXT                 NULL COLLATE 'utf8mb4_unicode_ci',
    `started_at`  DATETIME             NULL,
    `finished_at` DATETIME             NULL,
    `created_at`  DATETIME             NOT NULL,
    `updated_at`  DATETIME             NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE INDEX `execution_id` (`execution_id`) USING BTREE,
    INDEX `flow_id` (`flow_id`) USING BTREE,
    INDEX `state` (`state`) USING BTREE,
    INDEX `trigger_type` (`trigger_type`) USING BTREE,
    INDEX `created_at` (`created_at`) USING BTREE,
    CONSTRAINT `fk_executions_flow` FOREIGN KEY (`flow_id`) REFERENCES `flows` (`id`) ON DELETE CASCADE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

