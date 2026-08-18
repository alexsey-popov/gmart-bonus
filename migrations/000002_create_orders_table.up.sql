-- migrations/000002_create_orders_table.up.sql
-- Создание таблицы заказов
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    number VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL DEFAULT 'NEW',
    accrual numeric(10, 2) NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем комментарии к столбцам
COMMENT ON COLUMN orders.id IS 'Идентификатор заказа';
COMMENT ON COLUMN orders.user_id IS 'Идентификатор пользователя';
COMMENT ON COLUMN orders.number IS 'Номер заказа';
COMMENT ON COLUMN orders.status IS 'Статус заказа';
COMMENT ON COLUMN orders.accrual IS 'Количество бонусов за заказ';
COMMENT ON COLUMN orders.uploaded_at IS 'Момент поступления заказа в систему';

-- Индекс уникальности для номера заказа
CREATE UNIQUE INDEX idx_orders_number_unique ON orders(number);

-- Индекс для идентификатора пользователя
CREATE INDEX idx_orders_user_id ON orders(user_id);
-- Связь поля с идентификатором пользователя с таблицей пользователей + удаление всех заказов при удалении пользователя
ALTER TABLE orders
    ADD CONSTRAINT fk_orders_users
        FOREIGN KEY (user_id)
            REFERENCES users(id);