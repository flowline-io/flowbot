CREATE TABLE IF NOT EXISTS `agents`
(
	`id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
	`uid`          char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	`topic`        char(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	`hostid`       varchar(100) 											 NOT NULL,
	`hostname`     varchar(100) 											 NOT NULL,
	`online_duration`   int                                                  NOT NULL,
	`last_online_at`   datetime                                              NOT NULL,
	`created_at`   datetime                                                  NOT NULL,
	`updated_at`   datetime                                                  NOT NULL,
	PRIMARY KEY (`id`) USING BTREE,
	KEY `uid` (`uid`, `topic`)
)
COLLATE = 'utf8mb4_unicode_ci'
ENGINE = InnoDB;
