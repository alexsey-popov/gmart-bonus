package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/shopspring/decimal"
)

// CreateOrder Создание нового заказа.
// В случае ошибки уникальности по полю number возвращается дублирующая запись с ошибкой
func (rep Repository) CreateOrder(ctx context.Context, userId, number string) (model.Order, error) {
	// Создаём модифицированный model.Order
	// Поле IsNew даст нам понять перед нами новая запись или вернулась старая (более подробно в комментарии к запросу)
	order := struct {
		model.Order
		IsNew bool `db:"is_new"`
	}{}

	// В рамках одного запроса мы пытаемся создать запись и вернуть все её поля.
	// В случае конфликта по полю number мы делаем пустой update строки (обновим number на точно такое же значение).
	// Таким образом в случае конфликта уникальности нам не придётся делать ещё один запрос, чтобы понять
	// какому пользователю принадлежит указанный номер заказа.
	// Для того, чтобы отличить новую запись от обновлённой мы вводим поле xmax, при insert оно всегда равно 0,
	// а при update оно всегда больше нуля
	query := `INSERT INTO orders (user_id, number) 
		VALUES ($1, $2) 
		ON CONFLICT (number) DO UPDATE SET number = EXCLUDED.number
		RETURNING *, (xmax = 0) AS is_new;`

	// Делаем запрос
	err := rep.db.GetContext(ctx, &order, query, userId, number)
	if err != nil {
		rep.log.Error("ошибка при создании нового заказа", slog.Any("error", err))
		return model.Order{}, err
	}

	// Если нам вернулась ранее существующая строка - возвращаем её с ошибкой
	if !order.IsNew {
		return order.Order, ErrConflictUnique
	}

	return order.Order, nil
}

// GetUserOrders Получение списка заказов пользователя
func (rep Repository) GetUserOrders(ctx context.Context, userId string) (orders []model.Order, err error) {
	query := `SELECT * FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC LIMIT 100 `

	err = rep.db.SelectContext(ctx, &orders, query, userId)
	if err != nil {
		rep.log.Error("ошибка при получении списка заказов пользователя", slog.Any("error", err))
	}

	return
}

// GetUnfinishedOrders Необработанные заказы
func (rep Repository) GetUnfinishedOrders(ctx context.Context) (orders []model.Order, err error) {
	query := `SELECT * FROM orders WHERE status != ALL($1) ORDER BY uploaded_at LIMIT 100`

	err = rep.db.SelectContext(ctx, &orders, query, model.FinalOrderStatuses)
	if err != nil {
		rep.log.Error("ошибка при получении списка необработанных заказов", slog.Any("error", err))
	}

	return
}

// UpdateOrderStatus Обновление статуса наряда
func (rep Repository) UpdateOrderStatus(
	ctx context.Context,
	order model.Order,
	status model.OrderStatus,
	accrual *decimal.NullDecimal,
) (model.Order, error) {
	// Если статус не изменился - ничего не меняем
	if order.Status == status {
		return order, errors.New("статус заказ не изменился")
	}

	if order.Status.IsFinal() {
		return order, errors.New("заказ уже находится в окончательном статусе")
	}

	// начинаем транзакцию
	tx, err := rep.db.Begin()
	if err != nil {
		err = fmt.Errorf("ошибка при создании транзакции %w", err)
		rep.log.Error(err.Error(), slog.Any("error", err))

		return order, err
	}

	// Запрос 1 - Обновляем данные заказа в БД
	query := "UPDATE orders SET status=$1, accrual=$2 where id=$3"
	_, err = tx.ExecContext(ctx, query, status, accrual, order.Id)
	if err != nil {
		if errTx := tx.Rollback(); errTx != nil {
			rep.log.Error("ошибка при откате транзакции",
				slog.Any("error", err),
			)
		}
		return order, err
	}

	// Если заказ успешно обработан и сумма бонусов больше нуля - обновляем баланс пользователя
	if status == model.OrderStatusProcessed {
		// Запрос 2 - Исправляем баланс пользователя
		query = `UPDATE users SET current=COALESCE(current, 0.00)+COALESCE($1, 0.00)::numeric where id=$2`
		_, err = tx.ExecContext(ctx, query, accrual, order.UserId)
		if err != nil {
			rep.log.Error("ошибка при обновлении баланса пользователя",
				slog.Any("error", err),
			)

			if errTx := tx.Rollback(); errTx != nil {
				rep.log.Error("ошибка при откате транзакции",
					slog.Any("error", err),
				)
			}

			return order, err
		}

		// Обновляем данные в модели
		order.Status = status
		order.Accrual = accrual
	}

	rep.log.Info("Успешное обновление статуса в договоре",
		slog.String("order", order.Number),
	)

	// Применяем транзакцию
	if errTx := tx.Commit(); errTx != nil {
		rep.log.Error("ошибка при применении транзакции",
			slog.Any("error", err),
		)

		return order, errTx
	}

	return order, nil
}
