CREATE TABLE IF NOT EXISTS `messages`
(
    `id`              bigint                                                       NOT NULL AUTO_INCREMENT,
    `flag`            char(36) COLLATE utf8mb4_unicode_ci                          NOT NULL,
    `platform_id`     bigint                                                       NOT NULL DEFAULT (0),
    `platform_msg_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `topic`           char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `content`         json                                                                  DEFAULT NULL,
    `state`           tinyint                                                      NOT NULL,
    `created_at`      datetime                                                     NOT NULL,
    `updated_at`      datetime                                                     NOT NULL,
    `deleted_at`      datetime                                                              DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`),
    KEY `topic` (`topic`),
    KEY `platform` (`platform_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;