CREATE TABLE IF NOT EXISTS `fileuploads`
(
    `id`         bigint unsigned                                                NOT NULL AUTO_INCREMENT,
    `uid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `fid`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci      NOT NULL,
    `name`       varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `mimetype`   varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
    `size`       bigint                                                         NOT NULL,
    `location`   varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      int                                                            NOT NULL,
    `created_at` datetime                                                       NOT NULL,
    `updated_at` datetime                                                       NOT NULL,
    PRIMARY KEY (`id`),
    KEY `fileuploads_status` (`state`) USING BTREE,
    KEY `user_id` (`uid`) USING BTREE,
    KEY `file_id` (`fid`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
