package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

// ErrInsufficientFunds Ошибка при недостаточном количестве средств на бонусном счёте
var ErrInsufficientFunds = errors.New("Недостаточно средств")

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

// CreateWithdrawal Списание бонусов
func (rep Repository) CreateWithdrawal(ctx context.Context, userId, order string, sum decimal.Decimal) (balance model.Balance, err error) {
	// Получаем баланс пользователя
	balance, err = rep.GetUserBalance(ctx, userId)
	if err != nil {
		return balance, err
	}

	// Если у пользователя на счёте меньше баллов, чем он хочет списать - выдаём ошибку "Недостаточно средств"
	if balance.Current.LessThan(sum) {
		return balance, ErrInsufficientFunds
	}

	// Исправляем балансы пользователя ()
	balance.Current = balance.Current.Sub(sum)
	balance.Withdrawn = balance.Withdrawn.Add(sum)

	// начинаем транзакцию
	tx, err := rep.db.Begin()
	if err != nil {
		rep.log.Error("ошибка при создании транзакции",
			slog.Any("error", err),
		)

		return balance, err
	}

	// Всего в транзакции будут 2 запроса: Запись в withdrawals и изменение users

	// Запрос 1 - делаем запись в withdrawals
	query := `INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`
	_, err = tx.ExecContext(ctx, query, userId, order, sum)
	if err != nil {
		// Если произошла ошибка при выполнении запроса - пытаемся её классифицировать
		// Если произошла ошибка уникальности по полю order_number - отдаём соответствующую ошибку
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == "idx_withdrawals_order_number_unique" {
			if errTx := tx.Rollback(); errTx != nil {
				rep.log.Error("ошибка при откате транзакции",
					slog.Any("error", err),
				)
			}
			return balance, ErrConflictUnique
		}

		rep.log.Error("ошибка добавлении записи в withdrawals",
			slog.Any("error", err),
			slog.String("order", order),
			slog.String("sum", sum.String()),
		)

		if errTx := tx.Rollback(); errTx != nil {
			rep.log.Error("ошибка при откате транзакции",
				slog.Any("error", err),
			)
		}

		return balance, err
	}

	// Запрос 2 - исправляем баланс пользователя
	query = `UPDATE users SET current=$1, withdrawn=$2 where id=$3`
	_, err = tx.ExecContext(ctx, query, balance.Current, balance.Withdrawn, userId)
	if err != nil {
		rep.log.Error("ошибка при обновлении баланса пользователя",
			slog.Any("error", err),
			slog.String("current", balance.Current.String()),
			slog.String("withdrawn", balance.Withdrawn.String()),
		)

		if errTx := tx.Rollback(); errTx != nil {
			rep.log.Error("ошибка при откате транзакции",
				slog.Any("error", err),
			)
		}

		return balance, err
	}

	// Применяем транзакцию
	if errTx := tx.Commit(); errTx != nil {
		rep.log.Error("ошибка при применении транзакции",
			slog.Any("error", err),
		)

		return balance, errTx
	}

	return balance, nil
}

// GetUserOrders Получение списка заказов пользователя
func (rep Repository) GetUserWithdrawals(ctx context.Context, userId string) (orders []model.Withdrawal, err error) {
	query := `SELECT * FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC LIMIT 100 `

	err = rep.db.SelectContext(ctx, &orders, query, userId)
	if err != nil {
		rep.log.Error("ошибка при получении списка списаний пользователя", slog.Any("error", err))
	}

	return
}
