ALTER TABLE programs
  DROP FOREIGN KEY fk_program_broadcaster,
  DROP COLUMN broadcaster_id;
