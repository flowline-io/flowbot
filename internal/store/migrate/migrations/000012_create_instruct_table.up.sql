CREATE TABLE IF NOT EXISTS `instruct`
(
    `id`         bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `no`         char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `object`     varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `bot`        varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `flag`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `content`    json                                                         NOT NULL,
    `priority`   int                                                          NOT NULL,
    `state`      tinyint                                                      NOT NULL,
    `expire_at`  datetime                                                     NOT NULL,
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`),
    KEY `no` (`no`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;