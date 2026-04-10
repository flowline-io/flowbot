CREATE TABLE IF NOT EXISTS `steps`
(
    `id`         BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uid`        CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `topic`      CHAR(36)            NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `job_id`     BIGINT(19)          NOT NULL DEFAULT '0',
    `action`     JSON                NOT NULL,
    `name`       VARCHAR(100)        NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `describe`   VARCHAR(300)        NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `node_id`    VARCHAR(50)         NOT NULL DEFAULT '' COLLATE 'utf8mb4_unicode_ci',
    `depend`     JSON                NULL     DEFAULT NULL,
    `input`      JSON                NULL     DEFAULT NULL,
    `output`     JSON                NULL     DEFAULT NULL,
    `error`      VARCHAR(1000)       NULL     DEFAULT NULL COLLATE 'utf8mb4_unicode_ci',
    `state`      TINYINT(3)          NOT NULL,
    `started_at` DATETIME            NULL     DEFAULT NULL,
    `ended_at`   DATETIME            NULL     DEFAULT NULL,
    `created_at` DATETIME            NOT NULL,
    `updated_at` DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `uid` (`uid`, `topic`) USING BTREE,
    INDEX `job_id` (`job_id`) USING BTREE,
    INDEX `node_id` (`node_id`) USING BTREE,
    INDEX `state` (`state`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;