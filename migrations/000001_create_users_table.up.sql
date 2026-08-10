-- migrations/000001_create_users_table.up.sql
-- Создание таблицы с пользователями
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    current numeric(10, 2) NOT NULL DEFAULT 0.00,
    withdrawn numeric(10, 2) NOT NULL DEFAULT 0.00
);

-- Добавляем комментарии к столбцам
COMMENT ON COLUMN users.id IS 'Идентификатор пользователя';
COMMENT ON COLUMN users.login IS 'Логин';
COMMENT ON COLUMN users.password IS 'Пароль';
COMMENT ON COLUMN users.current IS 'Текущее количество бонусов';
COMMENT ON COLUMN users.withdrawn IS 'Общее количество потраченных бонусов';

-- Индекс уникальности для префикса
CREATE UNIQUE INDEX idx_users_login_unique ON users(login);