CREATE TABLE `chatbot_cycles`
(
    `id`         INT(10)    NOT NULL AUTO_INCREMENT,
    `uid`        CHAR(25)   NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`      CHAR(25)   NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `objectives` JSON       NOT NULL,
    `start_date` DATE       NOT NULL,
    `end_date`   DATE       NOT NULL,
    `state`      TINYINT(3) NOT NULL,
    `created_at` DATETIME   NOT NULL,
    `updated_at` DATETIME   NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid_topic` (`uid`, `topic`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;

CREATE TABLE `chatbot_reviews`
(
    `id`           INT(10)    NOT NULL AUTO_INCREMENT,
    `uid`          CHAR(25)   NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`        CHAR(25)   NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `objective_id` INT(10)    NOT NULL,
    `type`         TINYINT(3) NOT NULL,
    `rating`       TINYINT(3) NOT NULL,
    `created_at`   DATETIME   NOT NULL,
    `updated_at`   DATETIME   NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid_topic` (`uid`, `topic`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;


CREATE TABLE `chatbot_review_evaluations`
(
    `id`         INT(10)      NOT NULL AUTO_INCREMENT,
    `uid`        CHAR(25)     NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`      CHAR(25)     NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `review_id`  INT(10)      NOT NULL,
    `question`   VARCHAR(255) NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `reason`     VARCHAR(255) NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `solving`    VARCHAR(255) NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `created_at` DATETIME     NOT NULL,
    `updated_at` DATETIME     NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid_topic` (`uid`, `topic`) USING BTREE,
    INDEX `review_id` (`review_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;



CREATE TABLE `chatbot_key_result_values`
(
    `id`            int unsigned NOT NULL AUTO_INCREMENT,
    `key_result_id` int DEFAULT NULL,
    `value`         int          NOT NULL,
    `created_at`    datetime     NOT NULL,
    `updated_at`    datetime     NOT NULL,
    PRIMARY KEY (`id`),
    KEY `key_result_id` (`key_result_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_key_results`
(
    `id`            int unsigned                                                   NOT NULL AUTO_INCREMENT,
    `uid`           char(25) COLLATE utf8mb4_unicode_ci                            NOT NULL,
    `topic`         char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `objective_id`  int                                                            NOT NULL,
    `sequence`      int                                                            NOT NULL,
    `title`         varchar(100) COLLATE utf8mb4_unicode_ci                        NOT NULL,
    `memo`          varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `initial_value` int                                                            NOT NULL,
    `target_value`  int                                                            NOT NULL,
    `current_value` int                                                            NOT NULL,
    `value_mode`    tinyint                                                        NOT NULL,
    `tag`           varchar(100) COLLATE utf8mb4_unicode_ci                        NOT NULL,
    `created_at`    datetime                                                       NOT NULL,
    `updated_at`    datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_objectives`
(
    `id`            int unsigned                                                  NOT NULL AUTO_INCREMENT,
    `uid`           char(25) COLLATE utf8mb4_unicode_ci                           NOT NULL,
    `topic`         char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `sequence`      int                                                           NOT NULL,
    `title`         varchar(100) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `memo`          varchar(1000) COLLATE utf8mb4_unicode_ci                      NOT NULL,
    `motive`        varchar(1000) COLLATE utf8mb4_unicode_ci                      NOT NULL,
    `feasibility`   varchar(1000) COLLATE utf8mb4_unicode_ci                      NOT NULL,
    `is_plan`       tinyint                                                       NOT NULL,
    `plan_start`    bigint                                                        NOT NULL,
    `plan_end`      bigint                                                        NOT NULL,
    `total_value`   int                                                           NOT NULL,
    `current_value` int                                                           NOT NULL,
    `tag`           varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_data`  datetime                                                      NOT NULL,
    `updated_date`  datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_todos`
(
    `id`                int unsigned                                                  NOT NULL AUTO_INCREMENT,
    `uid`               char(25) COLLATE utf8mb4_unicode_ci                           NOT NULL,
    `topic`             char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `key_result_id`     INT(10)                                                       NOT NULL DEFAULT '0',
    `parent_id`         INT(10)                                                       NOT NULL DEFAULT '0',
    `sequence`          int                                                           NOT NULL,
    `content`           varchar(1000) COLLATE utf8mb4_unicode_ci                      NOT NULL,
    `category`          varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `remark`            varchar(100) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `priority`          int                                                           NOT NULL,
    `is_remind_at_time` tinyint                                                       NOT NULL,
    `remind_at`         bigint                                                        NOT NULL,
    `repeat_method`     varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `repeat_rule`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `repeat_end_at`     bigint                                                        NOT NULL,
    `complete`          tinyint                                                       NOT NULL,
    `created_at`        datetime                                                      NOT NULL,
    `updated_at`        datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`),
    INDEX `key_result_id` (`parent_id`) USING BTREE,
    INDEX `parent_id` (`parent_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
