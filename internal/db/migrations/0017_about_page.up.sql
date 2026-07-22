CREATE TABLE about_page_banner (
  id         TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  media_type ENUM('image','video') NOT NULL DEFAULT 'image',
  image_url  VARCHAR(1000) NULL,
  video_url  VARCHAR(1000) NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO about_page_banner (id) VALUES (1);

CREATE TABLE about_page_segments (
  id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  segment    ENUM('profile','music','audience') NOT NULL,
  title      VARCHAR(255) NOT NULL DEFAULT '',
  body       TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_segment (segment)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO about_page_segments (segment, title, body) VALUES
  ('profile', 'Profile', ''),
  ('music', 'Music', ''),
  ('audience', 'Audience', '');
