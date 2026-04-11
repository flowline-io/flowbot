CREATE TABLE IF NOT EXISTS `workflow_trigger`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `workflow_id` BIGINT(19)          NOT NULL DEFAULT '0',
    `type`        VARCHAR(20)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `rule`        JSON                NULL     DEFAULT NULL,
    `count`       INT(10)             NOT NULL DEFAULT '0',
    `state`       TINYINT(3)          NOT NULL,
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;