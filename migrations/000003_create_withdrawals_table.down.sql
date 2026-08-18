-- migrations/000003_create_withdrawals_table.down.sql
-- Откат создания таблицы списаний
ALTER TABLE withdrawals DROP CONSTRAINT fk_withdrawals_users;
DROP INDEX IF EXISTS idx_withdrawals_user_id;
DROP INDEX IF EXISTS idx_withdrawals_order_number_unique;

DROP TABLE IF EXISTS withdrawals;