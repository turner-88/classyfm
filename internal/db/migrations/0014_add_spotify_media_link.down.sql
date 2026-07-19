DELETE FROM media_links WHERE platform = 'spotify';

ALTER TABLE media_links MODIFY COLUMN platform ENUM('instagram','facebook','x','youtube') NOT NULL;
