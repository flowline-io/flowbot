CREATE TABLE IF NOT EXISTS `flow_queue_jobs`
(
    `id`           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `flow_id`      BIGINT(19)          NOT NULL,
    `trigger_type` VARCHAR(32)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `trigger_id`   VARCHAR(128)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `payload`      JSON                NULL     DEFAULT NULL,
    `status`       TINYINT(3)          NOT NULL DEFAULT 0,
    `attempts`     INT(11)             NOT NULL DEFAULT 0,
    `max_attempts` INT(11)             NOT NULL DEFAULT 3,
    `run_at`       DATETIME            NOT NULL,
    `locked_at`    DATETIME            NULL     DEFAULT NULL,
    `last_error`   TEXT                NULL     DEFAULT NULL,
    `created_at`   DATETIME            NOT NULL,
    `updated_at`   DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `idx_flow_queue_status_runat` (`status`, `run_at`) USING BTREE,
    INDEX `idx_flow_queue_flow_id` (`flow_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;
