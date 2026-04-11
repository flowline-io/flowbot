CREATE TABLE IF NOT EXISTS `dag`
(
    `id`             BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `workflow_id`    BIGINT(19)          NOT NULL DEFAULT '0',
    `script_id`      BIGINT(19)          NOT NULL,
    `script_version` SMALLINT(5)         NOT NULL,
    `nodes`          JSON                NOT NULL,
    `edges`          JSON                NOT NULL,
    `created_at`     DATETIME            NOT NULL,
    `updated_at`     DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;
