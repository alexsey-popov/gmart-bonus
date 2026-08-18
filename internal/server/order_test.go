package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrWithPause Проверка методов и конструктора ErrWithPause
func TestErrWithPause(t *testing.T) {
	t.Run("error message with inner error", func(t *testing.T) {
		err := NewErrWithPause(30)
		assert.Contains(t, err.Error(), "обработка приостановлена на")
		assert.Contains(t, err.Error(), "превышен лимит запросов")

		var pauseErr *ErrWithPause
		require.True(t, errors.As(err, &pauseErr))
		assert.Equal(t, 30*time.Second, pauseErr.Seconds)
		assert.NotNil(t, pauseErr.Unwrap())
	})

	t.Run("error message without inner error", func(t *testing.T) {
		err := &ErrWithPause{
			Seconds: 10 * time.Second,
			Err:     nil,
		}
		assert.Contains(t, err.Error(), "обработка приостановлена на")
		assert.Nil(t, err.Unwrap())
	})
}

// TestPauseOrderProcessing Проверка паузы обработки заказов
func TestPauseOrderProcessing(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := Server{log: log}

	t.Run("positive - pause completes normally", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		err := s.pauseOrderProcessing(ctx, 10*time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, time.Since(start), 5*time.Millisecond)
	})

	t.Run("negative - context cancelled during pause", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		err := s.pauseOrderProcessing(ctx, 1*time.Second)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
	})
}

// TestActualizeOrder Проверка актуализации данных заказа из системы расчёта бонусов
func TestActualizeOrder(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("final status order returns immediately", func(t *testing.T) {
		s := Server{log: log}
		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusProcessed,
		}
		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order, res)
	})

	t.Run("status OK updates order status and accrual", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := repository.NewRepository(sqlxDB, log)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/orders/12345678903", r.URL.Path)
			accrual := model.AccrualOrder{
				Order:  1234567890,
				Status: model.AccrualStatusProcessed,
				Accrual: &decimal.NullDecimal{
					Decimal: decimal.NewFromFloat(50),
					Valid:   true,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accrual)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
			rep: rep,
		}

		order := model.Order{
			Id:     "order-id-1",
			UserId: "user-id-1",
			Number: "12345678903",
			Status: model.OrderStatusProcessing,
		}

		nullDec := &decimal.NullDecimal{
			Decimal: decimal.NewFromFloat(50),
			Valid:   true,
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusProcessed, nullDec, order.Id).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE users SET current=COALESCE\(current, 0\.00\)\+COALESCE\(\$1, 0\.00\)::numeric where id=\$2`).
			WithArgs(nullDec, order.UserId).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order.Number, res.Number)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status OK with invalid JSON logs error and returns order", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order, res)
	})

	t.Run("status No Content updates order status to invalid", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := repository.NewRepository(sqlxDB, log)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
			rep: rep,
		}

		order := model.Order{
			Id:     "order-id-2",
			UserId: "user-id-2",
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusInvalid, &decimal.NullDecimal{}, order.Id).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order.Number, res.Number)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("status Too Many Requests with Retry-After header returns ErrWithPause", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "15")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		_, err := s.ActualizeOrder(context.Background(), order)
		require.Error(t, err)
		var pauseErr *ErrWithPause
		require.True(t, errors.As(err, &pauseErr))
		assert.Equal(t, 15*time.Second, pauseErr.Seconds)
	})

	t.Run("status Too Many Requests without valid Retry-After header defaults to 60", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		_, err := s.ActualizeOrder(context.Background(), order)
		require.Error(t, err)
		var pauseErr *ErrWithPause
		require.True(t, errors.As(err, &pauseErr))
		assert.Equal(t, 60*time.Second, pauseErr.Seconds)
	})

	t.Run("status Internal Server Error logs and returns no error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order, res)
	})

	t.Run("default status code logs and returns no error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		res, err := s.ActualizeOrder(context.Background(), order)
		require.NoError(t, err)
		assert.Equal(t, order, res)
	})

	t.Run("http request error returns error", func(t *testing.T) {
		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: "http://invalid-domain-name-that-does-not-exist-12345.local"},
		}

		order := model.Order{
			Number: "12345678903",
			Status: model.OrderStatusNew,
		}

		_, err := s.ActualizeOrder(context.Background(), order)
		require.Error(t, err)
	})
}

// TestStartOrderProcessing Проверка обработки пачки заказов пулом горутин
func TestStartOrderProcessing(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - processes orders successfully", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := repository.NewRepository(sqlxDB, log)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()

		s := Server{
			log: log,
			cfg: &config.Config{AccrualAddress: ts.URL},
			rep: rep,
		}

		orders := []model.Order{
			{Id: "id1", Number: "12345678903", Status: model.OrderStatusNew},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE orders SET status=\$1, accrual=\$2 where id=\$3`).
			WithArgs(model.OrderStatusInvalid, &decimal.NullDecimal{}, "id1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err = s.startOrderProcessing(context.Background(), rep, orders)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestLoopOrderProcessing Проверка циклической обработки заказов
func TestLoopOrderProcessing(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("context cancelled immediately", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s := Server{log: log}
		err := s.LoopOrderProcessing(ctx)
		require.NoError(t, err)
	})

	t.Run("db error in GetUnfinishedOrders returns error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := repository.NewRepository(sqlxDB, log)

		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))

		s := Server{
			log: log,
			rep: rep,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = s.LoopOrderProcessing(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
