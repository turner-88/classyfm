ALTER TABLE program_schedules
  ADD COLUMN broadcaster_id BIGINT UNSIGNED NULL AFTER host,
  ADD CONSTRAINT fk_schedule_broadcaster
    FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE SET NULL;
