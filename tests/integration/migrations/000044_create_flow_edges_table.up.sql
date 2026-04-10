CREATE TABLE IF NOT EXISTS `flow_edges`
(
    `id`            BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `flow_id`       BIGINT(20) UNSIGNED NOT NULL,
    `edge_id`       VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `source_node`   VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `target_node`   VARCHAR(64)         NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `source_port`   VARCHAR(64)         NULL COLLATE 'utf8mb4_unicode_ci',
    `target_port`   VARCHAR(64)         NULL COLLATE 'utf8mb4_unicode_ci',
    `label`         VARCHAR(255)        NULL COLLATE 'utf8mb4_unicode_ci',
    `created_at`    DATETIME            NOT NULL,
    `updated_at`    DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    INDEX `flow_id` (`flow_id`) USING BTREE,
    INDEX `source_node` (`source_node`) USING BTREE,
    INDEX `target_node` (`target_node`) USING BTREE,
    CONSTRAINT `fk_flow_edges_flow` FOREIGN KEY (`flow_id`) REFERENCES `flows` (`id`) ON DELETE CASCADE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

