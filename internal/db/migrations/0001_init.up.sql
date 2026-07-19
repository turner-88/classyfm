-- Initial ClassyFM schema.

CREATE TABLE news_items (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  source        ENUM('youtube','klikpositif','katasumbar','hot_release') NOT NULL,
  external_id   VARCHAR(255) NULL,
  title         VARCHAR(500) NOT NULL,
  slug          VARCHAR(255) NULL,
  excerpt       TEXT NULL,
  content       MEDIUMTEXT NULL,
  url           VARCHAR(1000) NULL,
  image_url     VARCHAR(1000) NULL,
  published_at  DATETIME NOT NULL,
  is_published  TINYINT(1) NOT NULL DEFAULT 1,
  is_featured   TINYINT(1) NOT NULL DEFAULT 0,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_source_external (source, external_id),
  UNIQUE KEY uq_slug (slug),
  KEY idx_feed (is_published, published_at),
  KEY idx_source (source, published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE programs (
  id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  title       VARCHAR(255) NOT NULL,
  slug        VARCHAR(255) NOT NULL UNIQUE,
  description TEXT NULL,
  host        VARCHAR(255) NULL,
  image_url   VARCHAR(1000) NULL,
  sort_order  INT NOT NULL DEFAULT 0,
  is_active   TINYINT(1) NOT NULL DEFAULT 1,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE program_schedules (
  id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  program_id  BIGINT UNSIGNED NOT NULL,
  day_of_week TINYINT NOT NULL,
  start_time  TIME NOT NULL,
  end_time    TIME NOT NULL,
  KEY idx_prog (program_id),
  KEY idx_day (day_of_week, start_time),
  CONSTRAINT fk_sched_prog FOREIGN KEY (program_id) REFERENCES programs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE media_links (
  id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  platform   ENUM('instagram','facebook','x','youtube') NOT NULL,
  label      VARCHAR(255) NULL,
  url        VARCHAR(1000) NOT NULL,
  handle     VARCHAR(255) NULL,
  embed_html TEXT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  is_active  TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE feed_sources (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  source          ENUM('youtube','klikpositif','katasumbar') NOT NULL UNIQUE,
  is_enabled      TINYINT(1) NOT NULL DEFAULT 1,
  endpoint        VARCHAR(1000) NULL,
  last_fetched_at DATETIME NULL,
  last_status     VARCHAR(255) NULL,
  item_count      INT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE settings (
  k VARCHAR(191) PRIMARY KEY,
  v TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE users (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  email         VARCHAR(191) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  role          ENUM('admin','editor') NOT NULL DEFAULT 'admin',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE sessions (
  token      CHAR(64) PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  expires_at DATETIME NOT NULL,
  KEY idx_exp (expires_at),
  CONSTRAINT fk_sess_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
