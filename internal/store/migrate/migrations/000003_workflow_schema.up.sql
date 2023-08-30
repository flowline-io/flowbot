CREATE TABLE `chatbot_dag`
(
    `id`          INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `workflow_id` INT(10)          NOT NULL,
    `nodes`       JSON             NOT NULL,
    `edges`       JSON             NOT NULL,
    `created_at`  DATETIME         NOT NULL,
    `updated_at`  DATETIME         NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;


CREATE TABLE `chatbot_jobs`
(
    `id`          INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `workflow_id` INT(10)          NOT NULL,
    `dag_id`      INT(10)          NOT NULL,
    `trigger_id`  INT(10)          NOT NULL,
    `state`       TINYINT(3)       NOT NULL,
    `started_at`  DATETIME         NULL DEFAULT NULL,
    `finished_at` DATETIME         NULL DEFAULT NULL,
    `created_at`  DATETIME         NOT NULL,
    `updated_at`  DATETIME         NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE,
    INDEX `state` (`state`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;


CREATE TABLE `chatbot_steps`
(
    `id`          INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `job_id`      INT(10)          NOT NULL,
    `action`      JSON             NOT NULL,
    `name`        VARCHAR(100)     NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `describe`    VARCHAR(300)     NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `node_id`     VARCHAR(50)      NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `depend`      JSON             NULL     DEFAULT NULL,
    `input`       JSON             NULL     DEFAULT NULL,
    `output`      JSON             NULL     DEFAULT NULL,
    `error`       VARCHAR(1000)    NULL     DEFAULT NULL COLLATE 'utf8mb4_unicode_ci',
    `state`       TINYINT(3)       NOT NULL,
    `started_at`  DATETIME         NULL     DEFAULT NULL,
    `finished_at` DATETIME         NULL     DEFAULT NULL,
    `created_at`  DATETIME         NOT NULL,
    `updated_at`  DATETIME         NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `job_id` (`job_id`) USING BTREE,
    INDEX `node_id` (`node_id`) USING BTREE,
    INDEX `state` (`state`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;


CREATE TABLE `chatbot_workflow`
(
    `id`               INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`              CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`            CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `flag`             CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `name`             VARCHAR(100)     NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `describe`         VARCHAR(300)     NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `successful_count` INT(10)          NOT NULL DEFAULT '0',
    `failed_count`     INT(10)          NOT NULL DEFAULT '0',
    `running_count`    INT(10)          NOT NULL DEFAULT '0',
    `canceled_count`   INT(10)          NOT NULL DEFAULT '0',
    `state`            TINYINT(3)       NOT NULL,
    `created_at`       DATETIME         NOT NULL,
    `updated_at`       DATETIME         NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `flag` (`flag`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;



CREATE TABLE `chatbot_workflow_trigger`
(
    `id`          INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`         CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`       CHAR(25)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `workflow_id` INT(10)          NOT NULL,
    `type`        VARCHAR(20)      NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `rule`        JSON             NOT NULL,
    `count`       INT(10)          NOT NULL DEFAULT '0',
    `state`       TINYINT(3)       NOT NULL,
    `created_at`  DATETIME         NOT NULL,
    `updated_at`  DATETIME         NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `workflow_id` (`workflow_id`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB
;
