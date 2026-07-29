ALTER TABLE programs ADD COLUMN host VARCHAR(255) NULL AFTER description;
ALTER TABLE program_schedules ADD COLUMN host VARCHAR(255) NULL AFTER end_time;
CREATE TABLE broadcaster_programs (
  broadcaster_id BIGINT UNSIGNED NOT NULL,
  program_id     BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (broadcaster_id, program_id),
  KEY idx_broadcaster_programs_program (program_id),
  CONSTRAINT fk_broadcaster_programs_broadcaster FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE CASCADE,
  CONSTRAINT fk_broadcaster_programs_program FOREIGN KEY (program_id) REFERENCES programs (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
