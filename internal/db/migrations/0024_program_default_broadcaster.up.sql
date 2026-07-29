-- A program-level default broadcaster. A schedule slot's effective broadcaster is
-- COALESCE(program_schedules.broadcaster_id, programs.broadcaster_id) - the slot may
-- still override, this only fills in the blanks.
ALTER TABLE programs
  ADD COLUMN broadcaster_id BIGINT UNSIGNED NULL AFTER image_url,
  ADD CONSTRAINT fk_program_broadcaster
    FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE SET NULL;
