CREATE TABLE IF NOT EXISTS `flow_nodes`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `flow_id`     BIGINT(20) UNSIGNED NOT NULL,
    `node_id`     VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `type`        VARCHAR(32)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `bot`         VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `rule_id`     VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `label`       VARCHAR(255)        NULL COLLATE 'utf8mb4_unicode_ci',
    `position_x`  INT(11)             NULL DEFAULT '0',
    `position_y`  INT(11)             NULL DEFAULT '0',
    `parameters`  JSON                NULL,
    `variables`   JSON                NULL,
    `conditions` JSON                NULL,
    `created_at` DATETIME            NOT NULL,
    `updated_at` DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `flow_id` (`flow_id`) USING BTREE,
    INDEX `node_id` (`node_id`) USING BTREE,
    INDEX `type` (`type`) USING BTREE,
    CONSTRAINT `fk_flow_nodes_flow` FOREIGN KEY (`flow_id`) REFERENCES `flows` (`id`) ON DELETE CASCADE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

