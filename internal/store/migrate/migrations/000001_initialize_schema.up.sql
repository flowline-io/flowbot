CREATE TABLE IF NOT EXISTS `action`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `seqid`      int                                                           NOT NULL,
    `value`      varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `behavior`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `count`      int                                                           NOT NULL,
    `extra`      json DEFAULT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `bots`
(
    `id`         bigint                                                       NOT NULL AUTO_INCREMENT,
    `name`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      tinyint                                                      NOT NULL DEFAULT (0),
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `configs`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `key`        varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `value`      json                                                          NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `counters`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `digit`      bigint                                                        NOT NULL,
    `status`     int                                                           NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `counter_records`
(
    `counter_id` bigint unsigned NOT NULL DEFAULT (0),
    `digit`      int             NOT NULL,
    `created_at` datetime        NOT NULL,
    PRIMARY KEY (`counter_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `cycles`
(
    `id`         bigint                                                    NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `objectives` json                                                      NOT NULL,
    `start_date` date                                                      NOT NULL,
    `end_date`   date                                                      NOT NULL,
    `state`      tinyint                                                   NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid_topic` (`uid`, `topic`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `dag`
(
    `id`          bigint unsigned                                           NOT NULL AUTO_INCREMENT,
    `uid`         char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `topic`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `workflow_id` bigint                                                    NOT NULL DEFAULT (0),
    `nodes`       json                                                      NOT NULL,
    `edges`       json                                                      NOT NULL,
    `created_at`  datetime                                                  NOT NULL,
    `updated_at`  datetime                                                  NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid` (`uid`, `topic`) USING BTREE,
    KEY `workflow_id` (`workflow_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `data`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `key`        varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `value`      json                                                          NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `fileuploads`
(
    `id`         bigint                                                         NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `name`       varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `mimetype`   varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `size`       bigint                                                         NOT NULL,
    `location`   varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      int                                                            NOT NULL,
    `created_at` datetime                                                       NOT NULL,
    `updated_at` datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `fileuploads_status` (`state`) USING BTREE,
    KEY `user_id` (`uid`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `form`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `form_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `schema`     json                                                          NOT NULL,
    `values`     json DEFAULT NULL,
    `extra`      json DEFAULT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `form_id` (`form_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



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



CREATE TABLE IF NOT EXISTS `jobs`
(
    `id`          bigint unsigned                                           NOT NULL AUTO_INCREMENT,
    `uid`         char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `topic`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `workflow_id` bigint                                                    NOT NULL DEFAULT (0),
    `dag_id`      bigint                                                    NOT NULL DEFAULT (0),
    `trigger_id`  bigint                                                    NOT NULL DEFAULT (0),
    `state`       tinyint                                                   NOT NULL,
    `started_at`  datetime                                                           DEFAULT NULL,
    `ended_at`    datetime                                                           DEFAULT NULL,
    `created_at`  datetime                                                  NOT NULL,
    `updated_at`  datetime                                                  NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid` (`uid`, `topic`) USING BTREE,
    KEY `workflow_id` (`workflow_id`) USING BTREE,
    KEY `state` (`state`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



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



CREATE TABLE IF NOT EXISTS `key_result_values`
(
    `id`            bigint unsigned NOT NULL AUTO_INCREMENT,
    `key_result_id` bigint                   DEFAULT NULL,
    `value`         int             NOT NULL,
    `memo`          VARCHAR(1000)   NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `created_at`    datetime        NOT NULL,
    `updated_at`    datetime        NOT NULL,
    PRIMARY KEY (`id`),
    KEY `key_result_id` (`key_result_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `messages`
(
    `id`              bigint                                                       NOT NULL AUTO_INCREMENT,
    `flag`            char(36) COLLATE utf8mb4_unicode_ci                          NOT NULL,
    `platform_id`     bigint                                                       NOT NULL DEFAULT (0),
    `platform_msg_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `topic`           char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `content`         json                                                                  DEFAULT NULL,
    `state`           tinyint                                                      NOT NULL,
    `created_at`      datetime                                                     NOT NULL,
    `updated_at`      datetime                                                     NOT NULL,
    `deleted_at`      datetime                                                              DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`),
    KEY `topic` (`topic`),
    KEY `platform` (`platform_id`) USING BTREE
) ENGINE = InnoDB
  AUTO_INCREMENT = 36
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `oauth`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `name`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `type`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `token`      varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `extra`      json                                                          NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE `objectives`
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



CREATE TABLE IF NOT EXISTS `pages`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `page_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `type`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `schema`     json                                                          NOT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `page_id` (`page_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 9
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



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



CREATE TABLE IF NOT EXISTS `pipelines`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`       char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `rule_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `version`    int                                                           NOT NULL,
    `stage`      int                                                           NOT NULL,
    `values`     json DEFAULT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `platforms`
(
    `id`         bigint      NOT NULL AUTO_INCREMENT,
    `name`       varchar(50) NOT NULL,
    `created_at` datetime    NOT NULL,
    `updated_at` datetime    NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE `platform_users`
(
    `id`          bigint                                                        NOT NULL AUTO_INCREMENT,
    `platform_id` bigint                                                        NOT NULL DEFAULT '0',
    `user_id`     bigint                                                        NOT NULL DEFAULT '0',
    `flag`        varchar(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `name`        varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `email`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `avatar_url`  varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `is_bot`      tinyint(1)                                                    NOT NULL DEFAULT '0',
    `created_at`  datetime                                                      NOT NULL,
    `updated_at`  datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `platform_id` (`platform_id`),
    KEY `user_id` (`user_id`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `reviews`
(
    `id`           bigint                                                    NOT NULL AUTO_INCREMENT,
    `uid`          char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `topic`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `objective_id` bigint                                                    NOT NULL DEFAULT (0),
    `type`         tinyint                                                   NOT NULL,
    `rating`       tinyint                                                   NOT NULL,
    `created_at`   datetime                                                  NOT NULL,
    `updated_at`   datetime                                                  NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid_topic` (`uid`, `topic`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `review_evaluations`
(
    `id`         bigint                                                        NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `review_id`  bigint                                                        NOT NULL DEFAULT (0),
    `question`   varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `reason`     varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `solving`    varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid_topic` (`uid`, `topic`) USING BTREE,
    KEY `review_id` (`review_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `schema_migrations`
(
    `version` int     NOT NULL AUTO_INCREMENT,
    `dirty`   tinyint NOT NULL DEFAULT (0),
    PRIMARY KEY (`version`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `session`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `rule_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `init`       json                                                          NOT NULL,
    `values`     json                                                          NOT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `steps`
(
    `id`         bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `job_id`     bigint                                                        NOT NULL DEFAULT (0),
    `action`     json                                                          NOT NULL,
    `name`       varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `describe`   varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    `node_id`    varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL DEFAULT '',
    `depend`     json                                                                   DEFAULT NULL,
    `input`      json                                                                   DEFAULT NULL,
    `output`     json                                                                   DEFAULT NULL,
    `error`      varchar(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci         DEFAULT NULL,
    `state`      tinyint                                                       NOT NULL,
    `started_at` datetime                                                               DEFAULT NULL,
    `ended_at`   datetime                                                               DEFAULT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid` (`uid`, `topic`) USING BTREE,
    KEY `job_id` (`job_id`) USING BTREE,
    KEY `node_id` (`node_id`),
    KEY `state` (`state`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



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



CREATE TABLE IF NOT EXISTS `topics`
(
    `id`         bigint                                                       NOT NULL AUTO_INCREMENT,
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
  AUTO_INCREMENT = 16
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



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



CREATE TABLE IF NOT EXISTS `users`
(
    `id`         bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `flag`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `name`       varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `tags`       json                                                                  DEFAULT NULL,
    `state`      smallint                                                     NOT NULL DEFAULT '0',
    `created_at` datetime                                                     NOT NULL,
    `updated_at` datetime                                                     NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `workflow`
(
    `id`               bigint unsigned                                               NOT NULL AUTO_INCREMENT,
    `uid`              char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`            char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`             char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `name`             varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `describe`         varchar(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `successful_count` int                                                           NOT NULL DEFAULT '0',
    `failed_count`     int                                                           NOT NULL DEFAULT '0',
    `running_count`    int                                                           NOT NULL DEFAULT '0',
    `canceled_count`   int                                                           NOT NULL DEFAULT '0',
    `state`            tinyint                                                       NOT NULL,
    `created_at`       datetime                                                      NOT NULL,
    `updated_at`       datetime                                                      NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid` (`uid`, `topic`) USING BTREE,
    KEY `flag` (`flag`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE IF NOT EXISTS `workflow_trigger`
(
    `id`          bigint unsigned                                              NOT NULL AUTO_INCREMENT,
    `uid`         char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `topic`       char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `workflow_id` bigint                                                       NOT NULL DEFAULT (0),
    `type`        varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `rule`        json                                                         NOT NULL,
    `count`       int                                                          NOT NULL DEFAULT '0',
    `state`       tinyint                                                      NOT NULL,
    `created_at`  datetime                                                     NOT NULL,
    `updated_at`  datetime                                                     NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `uid` (`uid`, `topic`) USING BTREE,
    KEY `workflow_id` (`workflow_id`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `platform_bots`
(
    `id`          BIGINT(19)  NOT NULL AUTO_INCREMENT,
    `platform_id` BIGINT(19)  NOT NULL DEFAULT '0',
    `bot_id`      BIGINT(19)  NOT NULL DEFAULT '0',
    `flag`        VARCHAR(50) NOT NULL DEFAULT '0' COLLATE 'utf8mb4_unicode_ci',
    `created_at`  DATETIME    NOT NULL,
    `updated_at`  DATETIME    NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `platform_id` (`platform_id`) USING BTREE,
    INDEX `bot_id` (`bot_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

CREATE TABLE `channels`
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

CREATE TABLE `platform_channels`
(
    `id`          BIGINT(19)  NOT NULL AUTO_INCREMENT,
    `platform_id` BIGINT(19)  NOT NULL DEFAULT '0',
    `channel_id`  BIGINT(19)  NOT NULL DEFAULT '0',
    `flag`        VARCHAR(50) NOT NULL DEFAULT '0' COLLATE 'utf8mb4_unicode_ci',
    `created_at`  DATETIME    NOT NULL,
    `updated_at`  DATETIME    NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `platform_id` (`platform_id`) USING BTREE,
    INDEX `channel_id` (`channel_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;
