CREATE TABLE IF NOT EXISTS `reviews`
(
    `id`           bigint unsigned                                           NOT NULL AUTO_INCREMENT,
    `uid`          char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `topic`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `objective_id` bigint                                                    NOT NULL DEFAULT (0),
    `type`         tinyint                                                   NOT NULL,
    `rating`       tinyint                                                   NOT NULL,
    `created_at`   datetime                                                  NOT NULL,
    `updated_at`   datetime                                                  NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid_topic` (`uid`, `topic`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;