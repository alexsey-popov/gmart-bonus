package repository

import (
	"context"
	"log/slog"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
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
