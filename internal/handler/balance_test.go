package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserBalance Проверка обработчика получения баланса пользователя
func TestGetUserBalance(t *testing.T) {
	t.Run("negative - неавторизованный запрос возвращает 401", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
		rec := httptest.NewRecorder()

		h.GetUserBalance(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("negative - ошибка репозитория возвращает 500", func(t *testing.T) {
		rep := &fakeRepository{
			getUserBalanceFunc: func(ctx context.Context, userId string) (model.Balance, error) {
				return model.Balance{}, errors.New("db error")
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserBalance(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("positive - успешное получение баланса возвращает 200 и json", func(t *testing.T) {
		rep := &fakeRepository{
			getUserBalanceFunc: func(ctx context.Context, userId string) (model.Balance, error) {
				assert.Equal(t, "user123", userId)
				return model.Balance{
					Current:   decimal.NewFromInt(500),
					Withdrawn: decimal.NewFromInt(100),
				}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserBalance(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"current":"500"`)
		assert.Contains(t, rec.Body.String(), `"withdrawn":"100"`)
	})
}

// TestCreateWithdrawal Проверка обработчика списания бонусов
func TestCreateWithdrawal(t *testing.T) {
	t.Run("negative - неавторизованный запрос возвращает 401", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"12345678903","sum":10}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("negative - некорректный JSON тела запроса возвращает 500", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`invalid json`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("negative - некорректный номер заказа по алгоритму Луна возвращает 422", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"order":"123456789030","sum":10}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Некорректный номер заказа")
	})

	t.Run("negative - ошибка валидации (пропуск обязательного поля) возвращает 400", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"sum":10}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - сумма бонусов <= 0 возвращает 422", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"order":"12345678903","sum":0}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Сумма бонусов к списанию должна быть больше нуля")
	})

	t.Run("negative - недостаточно средств на счете (ErrInsufficientFunds) возвращает 402", func(t *testing.T) {
		rep := &fakeRepository{
			createWithdrawalFunc: func(ctx context.Context, userId, order string, sum decimal.Decimal) (model.Balance, error) {
				return model.Balance{}, repository.ErrInsufficientFunds
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"order":"12345678903","sum":1000}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusPaymentRequired, rec.Code)
		assert.Contains(t, rec.Body.String(), "Недостаточно бонусов на счёте")
	})

	t.Run("negative - ошибка репозитория возвращает 500", func(t *testing.T) {
		rep := &fakeRepository{
			createWithdrawalFunc: func(ctx context.Context, userId, order string, sum decimal.Decimal) (model.Balance, error) {
				return model.Balance{}, errors.New("db error")
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"order":"12345678903","sum":10}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("positive - успешное списание бонусов возвращает 200 и json", func(t *testing.T) {
		rep := &fakeRepository{
			createWithdrawalFunc: func(ctx context.Context, userId, order string, sum decimal.Decimal) (model.Balance, error) {
				assert.Equal(t, "user123", userId)
				assert.Equal(t, "12345678903", order)
				assert.Equal(t, decimal.NewFromInt(50), sum)
				return model.Balance{
					Current:   decimal.NewFromInt(450),
					Withdrawn: decimal.NewFromInt(150),
				}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader(`{"order":"12345678903","sum":50}`))
		rec := httptest.NewRecorder()

		h.CreateWithdrawal(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"current":"450"`)
		assert.Contains(t, rec.Body.String(), `"withdrawn":"150"`)
	})
}

// TestGetUserWithdrawals Проверка обработчика истории списаний бонусов
func TestGetUserWithdrawals(t *testing.T) {
	t.Run("negative - неавторизованный запрос возвращает 401", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
		rec := httptest.NewRecorder()

		h.GetUserWithdrawals(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("negative - ошибка репозитория возвращает 500", func(t *testing.T) {
		rep := &fakeRepository{
			getUserWithdrawalsFunc: func(ctx context.Context, userId string) ([]model.Withdrawal, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserWithdrawals(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("positive - пустой список списаний возвращает 204", func(t *testing.T) {
		rep := &fakeRepository{
			getUserWithdrawalsFunc: func(ctx context.Context, userId string) ([]model.Withdrawal, error) {
				return []model.Withdrawal{}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserWithdrawals(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("positive - список списаний возвращает 200 и json", func(t *testing.T) {
		now := time.Now().UTC()
		nullDec := decimal.NewNullDecimal(decimal.NewFromInt(100))
		rep := &fakeRepository{
			getUserWithdrawalsFunc: func(ctx context.Context, userId string) ([]model.Withdrawal, error) {
				assert.Equal(t, "user123", userId)
				return []model.Withdrawal{
					{
						Id:          "1",
						UserId:      userId,
						OrderNumber: "12345678903",
						Sum:         &nullDec,
						ProcessedAt: now,
					},
				}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserWithdrawals(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"order":"12345678903"`)
		assert.Contains(t, rec.Body.String(), `"sum":"100"`)
	})
}
