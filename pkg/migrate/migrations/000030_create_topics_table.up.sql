CREATE TABLE IF NOT EXISTS `topics`
(
    `id`         bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `flag`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL DEFAULT '',
    `platform`   varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `owner`      bigint                                                       NOT NULL DEFAULT '0',
    `name`       char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `type`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `tags`       json                                                                  DEFAULT NULL,
    `state`      smallint                                                     NOT NULL DEFAULT '0',
    `touched_at` datetime                                                              DEFAULT NULL,
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`),
    KEY `topics_owner` (`owner`),
    KEY `platform` (`platform`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;