UPDATE users SET role = 'admin' WHERE role = 'superadmin';
ALTER TABLE users MODIFY role ENUM('admin','editor') NOT NULL DEFAULT 'admin';
