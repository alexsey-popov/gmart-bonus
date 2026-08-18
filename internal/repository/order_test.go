package repository

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateOrder Проверка создания заказа в репозитории
func TestCreateOrder(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное создание нового заказа", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		number := "12345678903"
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "is_new"}).
			AddRow("1", userId, number, model.OrderStatusNew, nil, now, true)

		mock.ExpectQuery(`INSERT INTO orders \(user_id, number\) VALUES \(\$1, \$2\) ON CONFLICT \(number\) DO UPDATE SET number = EXCLUDED\.number RETURNING \*, \(xmax = 0\) AS is_new;`).
			WithArgs(userId, number).
			WillReturnRows(rows)

		order, err := rep.CreateOrder(context.Background(), userId, number)
		require.NoError(t, err)
		assert.Equal(t, "1", order.Id)
		assert.Equal(t, userId, order.UserId)
		assert.Equal(t, number, order.Number)
		assert.Equal(t, model.OrderStatusNew, order.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - конфликт уникальности заказа (уже существует)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user456"
		number := "12345678903"
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "is_new"}).
			AddRow("1", "user123", number, model.OrderStatusNew, nil, now, false)

		mock.ExpectQuery(`INSERT INTO orders \(user_id, number\) VALUES \(\$1, \$2\) ON CONFLICT \(number\) DO UPDATE SET number = EXCLUDED\.number RETURNING \*, \(xmax = 0\) AS is_new;`).
			WithArgs(userId, number).
			WillReturnRows(rows)

		order, err := rep.CreateOrder(context.Background(), userId, number)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrConflictUnique))
		assert.Equal(t, "1", order.Id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка БД при создании заказа", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		number := "12345678903"
		dbErr := errors.New("db error")

		mock.ExpectQuery(`INSERT INTO orders \(user_id, number\) VALUES \(\$1, \$2\) ON CONFLICT \(number\) DO UPDATE SET number = EXCLUDED\.number RETURNING \*, \(xmax = 0\) AS is_new;`).
			WithArgs(userId, number).
			WillReturnError(dbErr)

		order, err := rep.CreateOrder(context.Background(), userId, number)
		require.Error(t, err)
		assert.Empty(t, order.Id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetUserOrders Проверка получения списка заказов пользователя
func TestGetUserOrders(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное получение списка заказов", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow("1", userId, "12345678903", model.OrderStatusNew, nil, now)

		mock.ExpectQuery(`SELECT \* FROM orders WHERE user_id = \$1 ORDER BY uploaded_at DESC LIMIT 100`).
			WithArgs(userId).
			WillReturnRows(rows)

		orders, err := rep.GetUserOrders(context.Background(), userId)
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Equal(t, "12345678903", orders[0].Number)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка БД при получении заказов", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		mock.ExpectQuery(`SELECT \* FROM orders WHERE user_id = \$1 ORDER BY uploaded_at DESC LIMIT 100`).
			WithArgs(userId).
			WillReturnError(errors.New("db error"))

		orders, err := rep.GetUserOrders(context.Background(), userId)
		require.Error(t, err)
		assert.Nil(t, orders)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetUnfinishedOrders Проверка получения необработанных заказов
func TestGetUnfinishedOrders(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное получение необработанных заказов", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		now := time.Now().UTC()
		rows := sqlmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at"}).
			AddRow("1", "user123", "12345678903", model.OrderStatusNew, nil, now)

		mock.ExpectQuery(`SELECT \* FROM orders WHERE status != ALL\(\$1\) ORDER BY uploaded_at LIMIT 100`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(rows)

		orders, err := rep.GetUnfinishedOrders(context.Background())
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Equal(t, model.OrderStatusNew, orders[0].Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка БД при получении необработанных заказов", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		mock.ExpectQuery(`SELECT \* FROM orders WHERE status != ALL\(\$1\) ORDER BY uploaded_at LIMIT 100`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		orders, err := rep.GetUnfinishedOrders(context.Background())
		require.Error(t, err)
		assert.Nil(t, orders)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateOrderStatus Проверка обновления статуса заказа
func TestUpdateOrderStatus(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("negative - статус заказа не изменился", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		ord := model.Order{
			Id:     "1",
			Status: model.OrderStatusNew,
		}

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusNew, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "статус заказ не изменился")
		assert.Equal(t, ord, updated)
	})

	t.Run("negative - заказ уже находится в окончательном статусе", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		ord := model.Order{
			Id:     "1",
			Status: model.OrderStatusProcessed,
		}

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusInvalid, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "заказ уже находится в окончательном статусе")
		assert.Equal(t, ord, updated)
	})

	t.Run("negative - ошибка начала транзакции (Begin)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		ord := model.Order{
			Id:     "1",
			Status: model.OrderStatusNew,
		}

		mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusProcessing, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при создании транзакции")
		assert.Equal(t, ord, updated)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка обновления заказа в транзакции вызывает rollback", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		ord := model.Order{
			Id:     "1",
			Status: model.OrderStatusNew,
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusInvalid, (*decimal.NullDecimal)(nil), "1").
			WillReturnError(errors.New("exec error"))
		mock.ExpectRollback()

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusInvalid, nil)
		require.Error(t, err)
		assert.Equal(t, ord, updated)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - ошибка обновления баланса при статусе PROCESSED вызывает rollback", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		ord := model.Order{
			Id:     "1",
			UserId: userId,
			Status: model.OrderStatusNew,
		}
		nullDec := decimal.NewNullDecimal(decimal.NewFromInt(100))
		accrualPtr := &nullDec

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusProcessed, accrualPtr, "1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(`UPDATE users SET current=COALESCE\(current, 0\.00\)\+COALESCE\(\$1, 0\.00\)::numeric where id=\$2`).
			WithArgs(accrualPtr, userId).
			WillReturnError(errors.New("balance update error"))

		mock.ExpectRollback()

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusProcessed, accrualPtr)
		require.Error(t, err)
		assert.Equal(t, ord, updated)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("positive - успешное обновление статуса (без обработки баланса)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		ord := model.Order{
			Id:     "1",
			Status: model.OrderStatusNew,
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusInvalid, (*decimal.NullDecimal)(nil), "1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusInvalid, nil)
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusInvalid, updated.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("positive - успешное обновление статуса на PROCESSED с обновлением баланса", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		userId := "user123"
		ord := model.Order{
			Id:     "1",
			UserId: userId,
			Status: model.OrderStatusNew,
		}
		nullDec := decimal.NewNullDecimal(decimal.NewFromInt(150))
		accrualPtr := &nullDec

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusProcessed, accrualPtr, "1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(`UPDATE users SET current=COALESCE\(current, 0\.00\)\+COALESCE\(\$1, 0\.00\)::numeric where id=\$2`).
			WithArgs(accrualPtr, userId).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		updated, err := rep.UpdateOrderStatus(context.Background(), ord, model.OrderStatusProcessed, accrualPtr)
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusProcessed, updated.Status)
		assert.Equal(t, accrualPtr, updated.Accrual)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
