-- Connect is the public chatroom (see /connect + the site-wide floating widget). Its
-- users authenticate via Google, so they cannot live in the admin `users` table
-- (password_hash NOT NULL, role ENUM fixed) - they get a dedicated `chat_users` table
-- keyed on the Google subject id. is_admin is a snapshot recomputed at every login by
-- matching the Google email against an active admin/superadmin row in `users`; it drives
-- the "special badge" and inline moderation controls. is_banned gates posting.
--
-- chat_messages stores posts; is_deleted is a soft-delete so admin moderation hides a
-- message without breaking the id sequence the polling delta (id > since) relies on. The
-- (is_deleted, id) index serves both the "recent visible" and "visible since id" reads.

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
