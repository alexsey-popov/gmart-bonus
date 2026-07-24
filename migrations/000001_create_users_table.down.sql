-- migrations/000001_create_users_table.down.sql
-- Откат создания таблицы пользователей
DROP INDEX IF EXISTS idx_users_login_unique;
DROP TABLE IF EXISTS users;