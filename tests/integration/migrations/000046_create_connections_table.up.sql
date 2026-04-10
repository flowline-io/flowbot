CREATE TABLE IF NOT EXISTS `connections`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `name`        VARCHAR(255)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `type`        VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `config`      JSON                NOT NULL,
    `enabled`     TINYINT(1)           NOT NULL DEFAULT '1',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `type` (`type`) USING BTREE,
    INDEX `enabled` (`enabled`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

