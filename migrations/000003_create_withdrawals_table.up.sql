-- migrations/000003_create_withdrawals_table.up.sql
-- Создание таблицы списаний
CREATE TABLE withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    order_number VARCHAR(255) NOT NULL,
    sum numeric(10, 2) NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем комментарии к столбцам
COMMENT ON COLUMN withdrawals.id IS 'Идентификатор заказа';
COMMENT ON COLUMN withdrawals.user_id IS 'Идентификатор пользователя';
COMMENT ON COLUMN withdrawals.order_number IS 'Номер заказа';
COMMENT ON COLUMN withdrawals.sum IS 'Сумма баллов к списанию';
COMMENT ON COLUMN withdrawals.processed_at IS 'Момент списания баллов';

-- Индекс уникальности для номера заказа
CREATE UNIQUE INDEX idx_withdrawals_order_number_unique ON withdrawals(order_number);

-- Индекс для идентификатора пользователя
CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);
-- Связь поля с идентификатором пользователя с таблицей пользователей + удаление всех заказов при удалении пользователя
ALTER TABLE withdrawals
    ADD CONSTRAINT fk_withdrawals_users
        FOREIGN KEY (user_id)
            REFERENCES users(id);