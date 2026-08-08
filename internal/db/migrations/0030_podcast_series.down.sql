ALTER TABLE podcasts
  DROP FOREIGN KEY fk_podcasts_series,
  DROP KEY idx_podcasts_series,
  DROP COLUMN series_id;

DROP TABLE podcast_series;
