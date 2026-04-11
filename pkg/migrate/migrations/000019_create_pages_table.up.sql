CREATE TABLE IF NOT EXISTS `pages`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `page_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `type`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `schema`     json                                                          NOT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `page_id` (`page_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;