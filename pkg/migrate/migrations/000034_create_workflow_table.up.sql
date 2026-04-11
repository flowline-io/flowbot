CREATE TABLE IF NOT EXISTS `workflow`
(
    `id`               BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`              CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`            CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `flag`             CHAR(25)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `name`             VARCHAR(100)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `describe`         VARCHAR(300)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `successful_count` INT(10)             NOT NULL DEFAULT '0',
    `failed_count`     INT(10)             NOT NULL DEFAULT '0',
    `running_count`    INT(10)             NOT NULL DEFAULT '0',
    `canceled_count`   INT(10)             NOT NULL DEFAULT '0',
    `state`            TINYINT(3)          NOT NULL,
    `created_at`       DATETIME            NOT NULL,
    `updated_at`       DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `flag` (`flag`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;