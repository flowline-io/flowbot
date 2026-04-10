CREATE TABLE IF NOT EXISTS `key_results`
(
    `id`            bigint unsigned                                                NOT NULL AUTO_INCREMENT,
    `uid`           char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `topic`         char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `objective_id`  bigint                                                         NOT NULL DEFAULT (0),
    `sequence`      int                                                            NOT NULL,
    `title`         varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `memo`          varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `initial_value` int                                                            NOT NULL,
    `target_value`  int                                                            NOT NULL,
    `current_value` int                                                            NOT NULL,
    `value_mode`    VARCHAR(20)                                                    NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `tag`           varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `created_at`    datetime                                                       NOT NULL,
    `updated_at`    datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;