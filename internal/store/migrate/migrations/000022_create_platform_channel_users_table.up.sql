CREATE TABLE `platform_channel_users`
(
    `id`           BIGINT(19) unsigned NOT NULL AUTO_INCREMENT,
    `platform_id`  BIGINT(19)          NOT NULL DEFAULT '0',
    `channel_flag` VARCHAR(50)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `user_flag`    VARCHAR(50)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `created_at`   DATETIME            NOT NULL,
    `updated_at`   DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `platform_id` (`platform_id`) USING BTREE,
    INDEX `channel_flag` (`channel_flag`) USING BTREE,
    INDEX `user_flag` (`user_flag`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;
