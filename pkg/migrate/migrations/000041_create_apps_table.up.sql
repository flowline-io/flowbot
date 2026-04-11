CREATE TABLE IF NOT EXISTS `apps`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
    `name`        VARCHAR(255)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `path`        VARCHAR(512)        NOT NULL COLLATE 'utf8mb4_unicode_ci',
    `container_id` VARCHAR(64)         NULL COLLATE 'utf8mb4_unicode_ci',
    `status`      VARCHAR(32)         NOT NULL DEFAULT 'unknown' COLLATE 'utf8mb4_unicode_ci',
    `docker_info` JSON                NULL,
    `created_at`  DATETIME            NOT NULL,
    `updated_at`  DATETIME            NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE INDEX `name` (`name`) USING BTREE,
    INDEX `status` (`status`) USING BTREE
)
    COLLATE = 'utf8mb4_unicode_ci'
    ENGINE = InnoDB;

