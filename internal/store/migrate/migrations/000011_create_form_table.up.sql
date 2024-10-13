CREATE TABLE IF NOT EXISTS `form`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `form_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `schema`     json                                                          NOT NULL,
    `values`     json DEFAULT NULL,
    `extra`      json DEFAULT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `form_id` (`form_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;