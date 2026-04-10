CREATE TABLE IF NOT EXISTS `platform_channels`
(
    `id`          BIGINT(19) unsigned NOT NULL AUTO_INCREMENT,
    `platform_id` BIGINT(19)          NOT NULL DEFAULT '0',
    `channel_id`  BIGINT(19)          NOT NULL DEFAULT '0',
    `flag`        VARCHAR(50)         NOT NULL DEFAULT '0' COLLATE 'utf8mb4_unicode_ci',
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `platform_id` (`platform_id`) USING BTREE,
    INDEX `channel_id` (`channel_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;