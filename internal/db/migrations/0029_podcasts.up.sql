-- Podcasts are admin-authored entries that point at a Spotify episode/show. Unlike
-- Hot Release (a source slice of news_items), they carry a broadcaster set, so they
-- get a dedicated table plus a podcast_broadcasters junction shaped exactly like
-- program_broadcasters (0026): composite PK, an index on the second FK, both FKs
-- ON DELETE CASCADE, and no sort_order column (names order by broadcasters.sort_order,
-- name wherever rendered).
--
-- thumb_url is resolved from spotify_url at save time via Spotify's oEmbed endpoint;
-- it is nullable because that resolution is best-effort and must not block the write.

CREATE TABLE podcasts (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  title        VARCHAR(500)  NOT NULL,
  slug         VARCHAR(255)  NOT NULL,
  description  TEXT          NOT NULL,
  spotify_url  VARCHAR(1000) NOT NULL,
  thumb_url    VARCHAR(1000) NULL,
  is_published TINYINT(1)    NOT NULL DEFAULT 1,
  created_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_podcasts_slug (slug),
  KEY idx_podcasts_published_created (is_published, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE podcast_broadcasters (
  podcast_id     BIGINT UNSIGNED NOT NULL,
  broadcaster_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (podcast_id, broadcaster_id),
  KEY idx_podcast_broadcasters_broadcaster (broadcaster_id),
  CONSTRAINT fk_podcast_broadcasters_podcast FOREIGN KEY (podcast_id) REFERENCES podcasts (id) ON DELETE CASCADE,
  CONSTRAINT fk_podcast_broadcasters_broadcaster FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
