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
