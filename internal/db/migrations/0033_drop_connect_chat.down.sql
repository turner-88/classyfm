-- Recreate the Connect chat schema from migration 0031 (rollback of the Firebase move).
-- Data is not restored.

CREATE TABLE chat_users (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  google_sub VARCHAR(191)  NOT NULL,
  email      VARCHAR(191)  NOT NULL,
  name       VARCHAR(255)  NOT NULL,
  avatar_url VARCHAR(1000) NULL,
  is_admin   TINYINT(1)    NOT NULL DEFAULT 0,
  is_banned  TINYINT(1)    NOT NULL DEFAULT 0,
  created_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_chat_users_google_sub (google_sub)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE chat_messages (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  chat_user_id BIGINT UNSIGNED NOT NULL,
  body         VARCHAR(1000) NOT NULL,
  is_deleted   TINYINT(1)    NOT NULL DEFAULT 0,
  created_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_chat_messages_visible (is_deleted, id),
  CONSTRAINT fk_chat_messages_user FOREIGN KEY (chat_user_id) REFERENCES chat_users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
