CREATE TABLE IF NOT EXISTS `flows`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `name`        VARCHAR(255)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `description` TEXT                NULL COLLATE 'utf8mb4_unicode_ci',
    `state`       TINYINT(3)          NOT NULL DEFAULT '0',
    `enabled`     TINYINT(1)           NOT NULL DEFAULT '1',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `state` (`state`) USING BTREE,
    INDEX `enabled` (`enabled`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

