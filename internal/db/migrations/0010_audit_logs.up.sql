CREATE TABLE audit_logs (
  id                   BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id              BIGINT UNSIGNED NULL,
  user_email_snapshot  VARCHAR(191) NOT NULL DEFAULT '',
  action               VARCHAR(32) NOT NULL,
  entity_type          VARCHAR(32) NOT NULL,
  entity_id            BIGINT UNSIGNED NULL,
  detail               TEXT NOT NULL,
  ip_address           VARCHAR(64) NOT NULL DEFAULT '',
  created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_audit_created_at (created_at),
  CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
