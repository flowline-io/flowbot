CREATE TABLE IF NOT EXISTS `parameter`
(
    `id`         bigint unsigned                                           NOT NULL AUTO_INCREMENT,
    `flag`       char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `params`     json DEFAULT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    `expired_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;