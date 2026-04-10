CREATE TABLE IF NOT EXISTS `platform_users`
(
    `id`          bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `platform_id` bigint                                                        NOT NULL DEFAULT '0',
    `user_id`     bigint                                                        NOT NULL DEFAULT '0',
    `flag`        varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `name`        varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `email`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `avatar_url`  varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `is_bot`      tinyint(1)                                                    NOT NULL DEFAULT '0',
    `created_at`  datetime                                                      NOT NULL,
    `updated_at`  datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `platform_id` (`platform_id`),
    KEY `user_id` (`user_id`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;