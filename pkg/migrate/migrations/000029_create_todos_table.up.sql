CREATE TABLE IF NOT EXISTS `todos`
(
    `id`                bigint unsigned                                                NOT NULL AUTO_INCREMENT,
    `uid`               char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `topic`             char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `key_result_id`     bigint                                                         NOT NULL DEFAULT (0),
    `parent_id`         bigint                                                         NOT NULL DEFAULT (0),
    `sequence`          int                                                            NOT NULL,
    `content`           varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `category`          varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `remark`            varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `priority`          int                                                            NOT NULL,
    `is_remind_at_time` tinyint                                                        NOT NULL,
    `remind_at`         bigint                                                         NOT NULL,
    `repeat_method`     varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `repeat_rule`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `repeat_end_at`     bigint                                                         NOT NULL,
    `complete`          tinyint                                                        NOT NULL,
    `created_at`        datetime                                                       NOT NULL,
    `updated_at`        datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`),
    KEY `key_result_id` (`parent_id`) USING BTREE,
    KEY `parent_id` (`parent_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;