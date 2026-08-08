-- Podcasts are organized into series (recurring shows). Series is an admin-managed
-- lookup, not a fixed enum, so it lives in its own table; every podcast references
-- exactly one series. Created and seeded here before the FK column is added so the
-- seeded id 1 backs the column default.

CREATE TABLE podcast_series (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name       VARCHAR(255) NOT NULL,
  slug       VARCHAR(255) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_podcast_series_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO podcast_series (name, slug, sort_order) VALUES
  ('Talkshow Bebas Pusing', 'talkshow-bebas-pusing', 1),
  ('Communitalk with Yeni Maiasnita', 'communitalk-with-yeni-maiasnita', 2),
  ('Special Talkshow', 'special-talkshow', 3);

-- DEFAULT 1 is a migration safety net (the table is empty, and the admin handler
-- always sets series_id explicitly). Default FK behavior is RESTRICT, so a series
-- in use by any podcast cannot be deleted.
ALTER TABLE podcasts
  ADD COLUMN series_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER slug,
  ADD KEY idx_podcasts_series (series_id),
  ADD CONSTRAINT fk_podcasts_series FOREIGN KEY (series_id) REFERENCES podcast_series (id);
