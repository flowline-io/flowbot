CREATE TABLE IF NOT EXISTS `authentications`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `connection_id` BIGINT(20) UNSIGNED NULL,
    `name`        VARCHAR(255)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `type`        VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `credentials` JSON                NOT NULL,
    `expires_at`  DATETIME            NULL,
    `enabled`     TINYINT(1)           NOT NULL DEFAULT '1',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `connection_id` (`connection_id`) USING BTREE,
    INDEX `type` (`type`) USING BTREE,
    INDEX `enabled` (`enabled`) USING BTREE,
    CONSTRAINT `fk_authentications_connection` FOREIGN KEY (`connection_id`) REFERENCES `connections` (`id`) ON DELETE SET NULL
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

