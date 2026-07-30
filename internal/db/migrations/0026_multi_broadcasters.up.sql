-- Programs and schedule slots go from one broadcaster to a set of them: shows are
-- co-hosted, and the scalar FKs added in 0022/0024 could not express that (0025 had
-- to drop every name but the first when backfilling).
--
-- The override stays, but at set level: a slot with any rows in schedule_broadcasters
-- uses exactly those, and a slot with none inherits the program's whole set from
-- program_broadcasters. That is the set version of the old
-- COALESCE(program_schedules.broadcaster_id, programs.broadcaster_id).
--
-- No sort_order column: names are ordered by broadcasters.sort_order, name wherever
-- they are rendered, same as the roster itself.

CREATE TABLE program_broadcasters (
  program_id     BIGINT UNSIGNED NOT NULL,
  broadcaster_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (program_id, broadcaster_id),
  KEY idx_program_broadcasters_broadcaster (broadcaster_id),
  CONSTRAINT fk_program_broadcasters_program FOREIGN KEY (program_id) REFERENCES programs (id) ON DELETE CASCADE,
  CONSTRAINT fk_program_broadcasters_broadcaster FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE schedule_broadcasters (
  schedule_id    BIGINT UNSIGNED NOT NULL,
  broadcaster_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (schedule_id, broadcaster_id),
  KEY idx_schedule_broadcasters_broadcaster (broadcaster_id),
  CONSTRAINT fk_schedule_broadcasters_schedule FOREIGN KEY (schedule_id) REFERENCES program_schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_schedule_broadcasters_broadcaster FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO program_broadcasters (program_id, broadcaster_id)
  SELECT id, broadcaster_id FROM programs WHERE broadcaster_id IS NOT NULL;

INSERT INTO schedule_broadcasters (schedule_id, broadcaster_id)
  SELECT id, broadcaster_id FROM program_schedules WHERE broadcaster_id IS NOT NULL;

ALTER TABLE programs
  DROP FOREIGN KEY fk_program_broadcaster,
  DROP COLUMN broadcaster_id;

ALTER TABLE program_schedules
  DROP FOREIGN KEY fk_schedule_broadcaster,
  DROP COLUMN broadcaster_id;
