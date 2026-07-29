-- migrations/000001_create_users_table.down.sql
-- Откат создания таблицы заказов
ALTER TABLE orders DROP CONSTRAINT fk_orders_users;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_orders_number_unique;

DROP TABLE IF EXISTS orders;