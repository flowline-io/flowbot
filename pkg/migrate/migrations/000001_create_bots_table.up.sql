CREATE TABLE IF NOT EXISTS `bots`
(
    `id`         bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `name`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      tinyint                                                      NOT NULL DEFAULT (0),
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;