CREATE TABLE IF NOT EXISTS `users`
(
    `id`         bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `flag`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `name`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `tags`       json                                                                  DEFAULT NULL,
    `state`      smallint                                                     NOT NULL DEFAULT '0',
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;