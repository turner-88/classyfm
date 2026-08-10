-- The Connect chat moved to Firebase Realtime Database (shared with the mobile app): the
-- website now reads/writes chat directly from the browser via the Firebase SDK, and chat
-- users authenticate with Firebase Authentication. The MySQL-backed chat is retired, so
-- these tables (and their sqlc queries) are dropped. The .down.sql recreates the schema
-- from migration 0031 for a clean rollback, but the message/user data is not restored.

DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_users;
