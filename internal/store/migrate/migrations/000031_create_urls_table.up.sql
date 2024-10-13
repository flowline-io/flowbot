CREATE TABLE IF NOT EXISTS `urls`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `flag`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `url`        varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      tinyint                                                       NOT NULL,
    `view_count` int                                                           NOT NULL DEFAULT '0',
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;