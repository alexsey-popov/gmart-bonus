package repository

import (
	"context"
	"log/slog"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
)

// GetUserBalance Получение баланса пользователя
func (rep Repository) GetUserBalance(ctx context.Context, userId string) (balance model.Balance, err error) {
	query := `SELECT current, withdrawn FROM users WHERE id = $1 LIMIT 1`

	err = rep.db.QueryRowContext(ctx, query, userId).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		rep.log.Error("ошибка при получении баланса пользователя",
			slog.Any("error", err),
			slog.String("user_id", userId),
		)
	}

	return
}
