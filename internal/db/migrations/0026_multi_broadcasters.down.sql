-- Lossy by nature: a scalar column can only keep one broadcaster per row, so the
-- lowest id wins and any co-hosts are dropped. The `host` column 0022 positioned
-- these AFTER no longer exists (dropped in 0023), hence the different placement.
ALTER TABLE programs
  ADD COLUMN broadcaster_id BIGINT UNSIGNED NULL AFTER image_url,
  ADD CONSTRAINT fk_program_broadcaster
    FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE SET NULL;

ALTER TABLE program_schedules
  ADD COLUMN broadcaster_id BIGINT UNSIGNED NULL AFTER end_time,
  ADD CONSTRAINT fk_schedule_broadcaster
    FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE SET NULL;

UPDATE programs p
  SET p.broadcaster_id = (
    SELECT MIN(pb.broadcaster_id) FROM program_broadcasters pb WHERE pb.program_id = p.id
  );

UPDATE program_schedules s
  SET s.broadcaster_id = (
    SELECT MIN(sb.broadcaster_id) FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id
  );

DROP TABLE schedule_broadcasters;
DROP TABLE program_broadcasters;
