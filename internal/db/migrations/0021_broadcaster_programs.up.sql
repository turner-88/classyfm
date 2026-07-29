-- Which programs a broadcaster presents. The existing programs.host and
-- program_schedules.host columns stay exactly as they are - they are free text
-- ("Andahayani, Yeni Maiasnita & Puti Adelya", "Hari"), often NULL on the slots
-- that actually air, and are what the schedule display and /live's announcer
-- badge read. They cannot be matched back to broadcaster rows, which is why the
-- relation is stored explicitly here rather than derived from them.
--
-- No backfill: a broadcaster with no rows here simply shows no Programs section.
CREATE TABLE broadcaster_programs (
  broadcaster_id BIGINT UNSIGNED NOT NULL,
  program_id     BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (broadcaster_id, program_id),
  KEY idx_broadcaster_programs_program (program_id),
  CONSTRAINT fk_broadcaster_programs_broadcaster FOREIGN KEY (broadcaster_id) REFERENCES broadcasters (id) ON DELETE CASCADE,
  CONSTRAINT fk_broadcaster_programs_program FOREIGN KEY (program_id) REFERENCES programs (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
