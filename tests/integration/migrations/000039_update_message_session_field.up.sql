ALTER TABLE `messages`
	ADD COLUMN `session` CHAR(36) NOT NULL AFTER `topic`;
