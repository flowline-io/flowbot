CREATE TABLE IF NOT EXISTS `objectives`
(
    `id`            bigint unsigned                                                NOT NULL AUTO_INCREMENT,
    `uid`           char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `topic`         char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `sequence`      int                                                            NOT NULL,
    `progress`      tinyint                                                        NOT NULL DEFAULT '0',
    `title`         varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `memo`          varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `motive`        varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `feasibility`   varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `is_plan`       tinyint                                                        NOT NULL DEFAULT '0',
    `plan_start`    date                                                           NOT NULL,
    `plan_end`      date                                                           NOT NULL,
    `total_value`   int                                                            NOT NULL,
    `current_value` int                                                            NOT NULL,
    `tag`           varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `created_data`  datetime                                                       NOT NULL,
    `updated_date`  datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;