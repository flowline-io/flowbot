CREATE TABLE IF NOT EXISTS `channels`
(
    `id`         bigint                                                       NOT NULL AUTO_INCREMENT,
    `name`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `flag`       varchar(36)                                                  NOT NULL,
    `state`      tinyint                                                      NOT NULL DEFAULT '0',
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;