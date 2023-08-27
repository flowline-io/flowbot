
CREATE TABLE kvmeta(
                       `key` VARCHAR(64),
                       createdat DATETIME(3),
                       `value` TEXT,
                       PRIMARY KEY(`key`),
                       INDEX kvmeta_createdat_key(createdat, `key`)
);

INSERT INTO kvmeta(`key`, `value`) VALUES("version", "100");

CREATE TABLE users(
                      id 			BIGINT NOT NULL,
                      createdat 	DATETIME(3) NOT NULL,
                      updatedat 	DATETIME(3) NOT NULL,
                      state 		SMALLINT NOT NULL DEFAULT 0,
                      stateat 	DATETIME(3),
                      access 		JSON,
                      lastseen 	DATETIME,
                      useragent 	VARCHAR(255) DEFAULT '',
                      public 		JSON,
                      tags		JSON, -- Denormalized array of tags

                      PRIMARY KEY(id),
                      INDEX users_state_stateat(state, stateat),
                      INDEX users_lastseen_updatedat(lastseen, updatedat)
);

# Indexed user tags.
CREATE TABLE usertags(
	id 		INT NOT NULL AUTO_INCREMENT,
	userid 	BIGINT NOT NULL,
	tag 	VARCHAR(96) NOT NULL,

	PRIMARY KEY(id),
	FOREIGN KEY(userid) REFERENCES users(id),
	INDEX usertags_tag(tag),
	UNIQUE INDEX usertags_userid_tag(userid, tag)
);

# Indexed devices. Normalized into a separate table.
CREATE TABLE devices(
	id 			INT NOT NULL AUTO_INCREMENT,
	userid 		BIGINT NOT NULL,
	hash 		CHAR(16) NOT NULL,
	deviceid 	TEXT NOT NULL,
	platform	VARCHAR(32),
	lastseen 	DATETIME NOT NULL,
	lang 		VARCHAR(8),

	PRIMARY KEY(id),
	FOREIGN KEY(userid) REFERENCES users(id),
	UNIQUE INDEX devices_hash(hash)
);

# Authentication records for the basic authentication scheme.
CREATE TABLE auth(
	id 		INT NOT NULL AUTO_INCREMENT,
	uname	VARCHAR(32) NOT NULL,
	userid 	BIGINT NOT NULL,
	scheme	VARCHAR(16) NOT NULL,
	authlvl	SMALLINT NOT NULL,
	secret 	VARCHAR(255) NOT NULL,
	expires DATETIME,

	PRIMARY KEY(id),
	FOREIGN KEY(userid) REFERENCES users(id),
	UNIQUE INDEX auth_userid_scheme(userid, scheme),
	UNIQUE INDEX auth_uname (uname)
);


# Topics
CREATE TABLE topics(
                       id			INT NOT NULL AUTO_INCREMENT,
                       createdat 	DATETIME(3) NOT NULL,
                       updatedat 	DATETIME(3) NOT NULL,
                       touchedat 	DATETIME(3),
                       state		SMALLINT NOT NULL DEFAULT 0,
                       stateat		DATETIME(3),
                       name		CHAR(25) NOT NULL,
                       usebt		TINYINT DEFAULT 0,
                       owner		BIGINT NOT NULL DEFAULT 0,
                       access		JSON,
                       seqid		INT NOT NULL DEFAULT 0,
                       delid		INT DEFAULT 0,
                       public		JSON,
                       tags		JSON, -- Denormalized array of tags

                       PRIMARY KEY(id),
                       UNIQUE INDEX topics_name (name),
                       INDEX topics_owner(owner),
                       INDEX topics_state_stateat(state, stateat)
);

# Indexed topic tags.
CREATE TABLE topictags(
	id 		INT NOT NULL AUTO_INCREMENT,
	topic 	CHAR(25) NOT NULL,
	tag 	VARCHAR(96) NOT NULL,

	PRIMARY KEY(id),
	FOREIGN KEY(topic) REFERENCES topics(name),
	INDEX topictags_tag (tag),
	UNIQUE INDEX topictags_userid_tag(topic, tag)
);

# Subscriptions
CREATE TABLE subscriptions(
                              id			INT NOT NULL AUTO_INCREMENT,
                              createdat	DATETIME(3) NOT NULL,
                              updatedat	DATETIME(3) NOT NULL,
                              deletedat	DATETIME(3),
                              userid		BIGINT NOT NULL,
                              topic		CHAR(25) NOT NULL,
                              delid		INT DEFAULT 0,
                              recvseqid	INT DEFAULT 0,
                              readseqid	INT DEFAULT 0,
                              modewant	CHAR(8),
                              modegiven	CHAR(8),
                              private		JSON,

                              PRIMARY KEY(id)	,
                              FOREIGN KEY(userid) REFERENCES users(id),
                              UNIQUE INDEX subscriptions_topic_userid(topic, userid),
                              INDEX subscriptions_topic(topic),
                              INDEX subscriptions_deletedat(deletedat)
);

# Messages
CREATE TABLE messages(
                         id 			INT NOT NULL AUTO_INCREMENT,
                         createdat 	DATETIME(3) NOT NULL,
                         updatedat 	DATETIME(3) NOT NULL,
                         deletedat 	DATETIME(3),
                         delid 		INT DEFAULT 0,
                         seqid 		INT NOT NULL,
                         topic 		CHAR(25) NOT NULL,
                         `from` 		BIGINT NOT NULL,
                         head 		JSON,
                         content 	JSON,

                         PRIMARY KEY(id),
                         FOREIGN KEY(topic) REFERENCES topics(name),
                         UNIQUE INDEX messages_topic_seqid (topic, seqid)
);

# Deletion log
CREATE TABLE dellog(
                       id			INT NOT NULL AUTO_INCREMENT,
                       topic		CHAR(25) NOT NULL,
                       deletedfor	BIGINT NOT NULL DEFAULT 0,
                       delid		INT NOT NULL,
                       low			INT NOT NULL,
                       hi			INT NOT NULL,

                       PRIMARY KEY(id),
                       FOREIGN KEY(topic) REFERENCES topics(name),
                       # For getting the list of deleted message ranges
                           INDEX dellog_topic_delid_deletedfor(topic,delid,deletedfor),
                       # Used when getting not-yet-deleted messages(messages LEFT JOIN dellog)
                           INDEX dellog_topic_deletedfor_low_hi(topic,deletedfor,low,hi),
                       # Used when deleting a user
                           INDEX dellog_deletedfor(deletedfor)
);

# User credentials
CREATE TABLE credentials(
                            id			INT NOT NULL AUTO_INCREMENT,
                            createdat	DATETIME(3) NOT NULL,
                            updatedat	DATETIME(3) NOT NULL,
                            deletedat	DATETIME(3),
                            method 		VARCHAR(16) NOT NULL,
                            value		VARCHAR(128) NOT NULL,
                            synthetic	VARCHAR(192) NOT NULL,
                            userid 		BIGINT NOT NULL,
                            resp		VARCHAR(255) NOT NULL,
                            done		TINYINT NOT NULL DEFAULT 0,
                            retries		INT NOT NULL DEFAULT 0,

                            PRIMARY KEY(id),
                            UNIQUE credentials_uniqueness(synthetic),
                            FOREIGN KEY(userid) REFERENCES users(id),
);

# Records of uploaded files. Files themselves are stored elsewhere.
CREATE TABLE fileuploads(
	id			BIGINT NOT NULL,
	createdat	DATETIME(3) NOT NULL,
	updatedat	DATETIME(3) NOT NULL,
	userid		BIGINT,
	status		INT NOT NULL,
	mimetype	VARCHAR(255) NOT NULL,
	size		BIGINT NOT NULL,
	location	VARCHAR(2048) NOT NULL,

	PRIMARY KEY(id),
	INDEX fileuploads_status(status)
);

# Links between uploaded files and messages or topics.
CREATE TABLE filemsglinks(
	id			INT NOT NULL AUTO_INCREMENT,
	createdat	DATETIME(3) NOT NULL,
	fileid		BIGINT NOT NULL,
	msgid		INT,
	topic		CHAR(25),
	userid		BIGINT,

	PRIMARY KEY(id),
	FOREIGN KEY(fileid) REFERENCES fileuploads(id) ON DELETE CASCADE,
	FOREIGN KEY(msgid) REFERENCES messages(id) ON DELETE CASCADE,
	FOREIGN KEY(topicid) REFERENCES topics(id) ON DELETE CASCADE,
	FOREIGN KEY(userid) REFERENCES users(id) ON DELETE CASCADE
);


CREATE TABLE `chatbot_action`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `seqid`      int                                                       NOT NULL,
    `value`      varchar(256) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `state`      tinyint                                                   NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_behavior`
(
    `id`         int unsigned                            NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`       varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
    `count`      int                                     NOT NULL,
    `extra`      json DEFAULT NULL,
    `created_at` datetime                                NOT NULL,
    `updated_at` datetime                                NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_configs`
(
    `id`         int unsigned                            NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(25) COLLATE utf8mb4_unicode_ci     NOT NULL,
    `key`        varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
    `value`      json                                    NOT NULL,
    `created_at` datetime                                NOT NULL,
    `updated_at` datetime                                NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_counters`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `flag`       varchar(100) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `digit`      bigint                                                    NOT NULL,
    `status`     int                                                       NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_data`
(
    `id`         int unsigned                            NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(25) COLLATE utf8mb4_unicode_ci     NOT NULL,
    `key`        varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
    `value`      json                                    NOT NULL,
    `created_at` datetime                                NOT NULL,
    `updated_at` datetime                                NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_form`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `form_id`    varchar(100) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `schema`     json                                                      NOT NULL,
    `values`     json DEFAULT NULL                                         NULL,
    `extra`      json DEFAULT NULL                                         NULL,
    `state`      tinyint                                                   NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `form_id` (`form_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;



CREATE TABLE `chatbot_oauth`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `name`       varchar(100) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `type`       varchar(50) COLLATE utf8mb4_unicode_ci                    NOT NULL,
    `token`      varchar(256) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `extra`      json                                                      NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_page`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `page_id`    varchar(100) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                       NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `type`       varchar(100) COLLATE utf8mb4_unicode_ci                   NOT NULL,
    `schema`     json                                                      NOT NULL,
    `state`      tinyint                                                   NOT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `page_id` (`page_id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_session`
(
    `id`         int unsigned                                                  NOT NULL AUTO_INCREMENT,
    `uid`        char(25) COLLATE utf8mb4_unicode_ci                           NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
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



CREATE TABLE `chatbot_url`
(
    `id`         int unsigned                            NOT NULL AUTO_INCREMENT,
    `flag`       varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
    `url`        varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
    `state`      tinyint                                 NOT NULL,
    `view_count` int                                     NOT NULL DEFAULT '0',
    `created_at` datetime                                NOT NULL,
    `updated_at` datetime                                NOT NULL,
    PRIMARY KEY (`id`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_counter_records`
(
    `counter_id` int unsigned NOT NULL AUTO_INCREMENT,
    `digit`      int          NOT NULL,
    `created_at` datetime     NOT NULL,
    PRIMARY KEY (`counter_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_instruct`
(
    `id`         int unsigned                                                 NOT NULL AUTO_INCREMENT,
    `no`         char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
    `uid`        char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci    NOT NULL,
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


CREATE TABLE `chatbot_pipelines`
(
    `id`         int unsigned                                                  NOT NULL AUTO_INCREMENT,
    `uid`        char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `topic`      char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `flag`       char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci     NOT NULL,
    `rule_id`    varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `version`    int                                                           NOT NULL,
    `stage`       int                                                           NOT NULL,
    `values`     json DEFAULT NULL,
    `state`      tinyint                                                       NOT NULL,
    `created_at` datetime                                                      NOT NULL,
    `updated_at` datetime                                                      NOT NULL,
    PRIMARY KEY (`id`),
    KEY `uid` (`uid`, `topic`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


CREATE TABLE `chatbot_parameter`
(
    `id`         int unsigned                                              NOT NULL AUTO_INCREMENT,
    `flag`       char(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    `params`     json DEFAULT NULL,
    `created_at` datetime                                                  NOT NULL,
    `updated_at` datetime                                                  NOT NULL,
    `expired_at` datetime                                                  NOT NULL,
    PRIMARY KEY (`id`),
    KEY `flag` (`flag`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
