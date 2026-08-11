ALTER TABLE podcast_series ADD COLUMN is_active TINYINT(1) NOT NULL DEFAULT 1 AFTER sort_order;
