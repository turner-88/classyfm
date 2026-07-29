ALTER TABLE program_schedules
  DROP FOREIGN KEY fk_schedule_broadcaster,
  DROP COLUMN broadcaster_id;
