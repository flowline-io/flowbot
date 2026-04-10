CREATE TABLE IF NOT EXISTS `jobs`
(
    `id`             BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`            CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`          CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `workflow_id`    BIGINT(19)          NOT NULL DEFAULT '0',
    `dag_id`         BIGINT(19)          NOT NULL DEFAULT '0',
    `trigger_id`     BIGINT(19)          NOT NULL DEFAULT '0',
    `script_version` SMALLINT(5)         NOT NULL DEFAULT '0',
    `state`          TINYINT(3)          NOT NULL,
    `started_at`     DATETIME            NULL     DEFAULT NULL,
    `ended_at`       DATETIME            NULL     DEFAULT NULL,
    `created_at`     DATETIME            NOT NULL,
    `updated_at`     DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE,
    INDEX `state` (`state`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;