ALTER TABLE users MODIFY role ENUM('admin','editor','superadmin') NOT NULL DEFAULT 'admin';
UPDATE users SET role = 'superadmin' WHERE role = 'admin';
UPDATE users SET role = 'admin' WHERE role = 'editor';
ALTER TABLE users MODIFY role ENUM('superadmin','admin') NOT NULL DEFAULT 'admin';
