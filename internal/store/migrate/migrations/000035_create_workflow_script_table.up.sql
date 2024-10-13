CREATE TABLE IF NOT EXISTS `workflow_script`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `workflow_id` BIGINT(20) UNSIGNED NOT NULL,
    `lang`        VARCHAR(10)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `code`        TEXT                NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `version`     SMALLINT(5)         NOT NULL DEFAULT '1',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;