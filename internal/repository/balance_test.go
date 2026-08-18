package repository

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserBalance Проверка получения баланса пользователя
func TestGetUserBalance(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное получение баланса", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "100.00"))

		balance, err := rep.GetUserBalance(context.Background(), userId)
		require.NoError(t, err)
		assert.True(t, balance.Current.Equal(decimal.NewFromInt(500)))
		assert.True(t, balance.Withdrawn.Equal(decimal.NewFromInt(100)))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка БД при получении баланса", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnError(sql.ErrNoRows)

		_, err = rep.GetUserBalance(context.Background(), userId)
		require.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetUserWithdrawals Проверка получения списка списаний пользователя
func TestGetUserWithdrawals(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное получение списка списаний", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).
			AddRow("1", userId, "12345678903", decimal.NewFromFloat(100), now)

		mock.ExpectQuery(`SELECT \* FROM withdrawals WHERE user_id = \$1 ORDER BY processed_at DESC LIMIT 100`).
			WithArgs(userId).
			WillReturnRows(rows)

		withdrawals, err := rep.GetUserWithdrawals(context.Background(), userId)
		require.NoError(t, err)
		require.Len(t, withdrawals, 1)
		assert.Equal(t, "12345678903", withdrawals[0].OrderNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка БД при получении списаний", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		mock.ExpectQuery(`SELECT \* FROM withdrawals WHERE user_id = \$1 ORDER BY processed_at DESC LIMIT 100`).
			WithArgs(userId).
			WillReturnError(errors.New("db error"))

		withdrawals, err := rep.GetUserWithdrawals(context.Background(), userId)
		require.Error(t, err)
		assert.Nil(t, withdrawals)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCreateWithdrawal Проверка списания бонусов
func TestCreateWithdrawal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("negative - ошибка получения баланса", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnError(sql.ErrNoRows)

		_, err = rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - недостаточно средств", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(200)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("100.00", "0.00"))

		balance, err := rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInsufficientFunds))
		assert.True(t, balance.Current.Equal(decimal.NewFromInt(100)))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка начала транзакции", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "0.00"))

		mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		_, err = rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "begin error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - конфликт уникальности номера заказа (UniqueViolation)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "0.00"))

		mock.ExpectBegin()

		pgErr := &pgconn.PgError{
			Code:           pgerrcode.UniqueViolation,
			ConstraintName: "idx_withdrawals_order_number_unique",
		}
		mock.ExpectExec(`INSERT INTO withdrawals \(user_id, order_number, sum\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs(userId, order, sum).
			WillReturnError(pgErr)

		mock.ExpectRollback()

		_, err = rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrConflictUnique))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка вставки в withdrawals вызывает rollback", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "0.00"))

		mock.ExpectBegin()

		mock.ExpectExec(`INSERT INTO withdrawals \(user_id, order_number, sum\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs(userId, order, sum).
			WillReturnError(errors.New("insert error"))

		mock.ExpectRollback()

		_, err = rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insert error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка обновления users вызывает rollback", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "0.00"))

		mock.ExpectBegin()

		mock.ExpectExec(`INSERT INTO withdrawals \(user_id, order_number, sum\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs(userId, order, sum).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(`UPDATE users SET current=\$1, withdrawn=\$2 where id=\$3`).
			WithArgs(decimal.NewFromFloat(400), decimal.NewFromFloat(100), userId).
			WillReturnError(errors.New("update error"))

		mock.ExpectRollback()

		_, err = rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("positive - успешное списание бонусов", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		order := "12345678903"
		sum := decimal.NewFromFloat(100)

		mock.ExpectQuery(`SELECT current, withdrawn FROM users WHERE id = \$1 LIMIT 1`).
			WithArgs(userId).
			WillReturnRows(sqlmock.NewRows([]string{"current", "withdrawn"}).AddRow("500.00", "0.00"))

		mock.ExpectBegin()

		mock.ExpectExec(`INSERT INTO withdrawals \(user_id, order_number, sum\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs(userId, order, sum).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(`UPDATE users SET current=\$1, withdrawn=\$2 where id=\$3`).
			WithArgs(decimal.NewFromFloat(400), decimal.NewFromFloat(100), userId).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		balance, err := rep.CreateWithdrawal(context.Background(), userId, order, sum)
		require.NoError(t, err)
		assert.True(t, balance.Current.Equal(decimal.NewFromInt(400)))
		assert.True(t, balance.Withdrawn.Equal(decimal.NewFromInt(100)))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
