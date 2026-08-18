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

// TestCreateOrder Проверка обработчика создания заказа
func TestCreateOrder(t *testing.T) {
	t.Run("negative - неавторизованный запрос возвращает 401", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("12345678903"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("negative - некорректный номер заказа по алгоритму Луна возвращает 422", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("123456789030")) // неверный Лун
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Некорректный номер заказа")
	})

	t.Run("negative - ошибка валидации (не число) возвращает 400", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("abc"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("positive - успешное создание заказа возвращает 202 и json", func(t *testing.T) {
		now := time.Now().UTC()
		rep := &fakeRepository{
			createOrderFunc: func(ctx context.Context, userId, number string) (model.Order, error) {
				assert.Equal(t, "user123", userId)
				assert.Equal(t, "12345678903", number)
				return model.Order{
					Number:     number,
					UserId:     userId,
					Status:     model.OrderStatusNew,
					UploadedAt: now,
				}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("12345678903"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		require.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"number":"12345678903"`)
		assert.Contains(t, rec.Body.String(), `"status":"NEW"`)
	})

	t.Run("positive - конфликт уникальности: заказ загружен тем же пользователем возвращает 200", func(t *testing.T) {
		now := time.Now().UTC()
		rep := &fakeRepository{
			createOrderFunc: func(ctx context.Context, userId, number string) (model.Order, error) {
				return model.Order{
					Number:     number,
					UserId:     "user123",
					Status:     model.OrderStatusProcessing,
					UploadedAt: now,
				}, repository.ErrConflictUnique
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("12345678903"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Номер заказа уже был загружен этим пользователем")
	})

	t.Run("negative - конфликт уникальности: заказ загружен другим пользователем возвращает 409", func(t *testing.T) {
		now := time.Now().UTC()
		rep := &fakeRepository{
			createOrderFunc: func(ctx context.Context, userId, number string) (model.Order, error) {
				return model.Order{
					Number:     number,
					UserId:     "other-user",
					Status:     model.OrderStatusProcessing,
					UploadedAt: now,
				}, repository.ErrConflictUnique
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("12345678903"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Номер заказа уже был загружен другим пользователем")
	})

	t.Run("negative - ошибка репозитория возвращает 500", func(t *testing.T) {
		rep := &fakeRepository{
			createOrderFunc: func(ctx context.Context, userId, number string) (model.Order, error) {
				return model.Order{}, errors.New("db error")
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		req.Method = http.MethodPost
		req.Body = io.NopCloser(strings.NewReader("12345678903"))
		rec := httptest.NewRecorder()

		h.CreateOrder(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// TestGetUserOrders Проверка обработчика получения списка заказов
func TestGetUserOrders(t *testing.T) {
	t.Run("negative - неавторизованный запрос возвращает 401", func(t *testing.T) {
		h := newTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
		rec := httptest.NewRecorder()

		h.GetUserOrders(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("negative - ошибка репозитория возвращает 500", func(t *testing.T) {
		rep := &fakeRepository{
			getUserOrdersFunc: func(ctx context.Context, userId string) ([]model.Order, error) {
				return nil, errors.New("db error")
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserOrders(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("positive - пустой список заказов возвращает 204", func(t *testing.T) {
		rep := &fakeRepository{
			getUserOrdersFunc: func(ctx context.Context, userId string) ([]model.Order, error) {
				return []model.Order{}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserOrders(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("positive - список заказов возвращает 200 и json", func(t *testing.T) {
		now := time.Now().UTC()
		nullDec := decimal.NewNullDecimal(decimal.NewFromInt(500))
		rep := &fakeRepository{
			getUserOrdersFunc: func(ctx context.Context, userId string) ([]model.Order, error) {
				assert.Equal(t, "user123", userId)
				return []model.Order{
					{
						Number:     "12345678903",
						UserId:     userId,
						Status:     model.OrderStatusProcessed,
						Accrual:    &nullDec,
						UploadedAt: now,
					},
				}, nil
			},
		}
		h := newTestHandler(t, rep)
		req := requestWithUserID(t, "user123")
		rec := httptest.NewRecorder()

		h.GetUserOrders(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"number":"12345678903"`)
		assert.Contains(t, rec.Body.String(), `"status":"PROCESSED"`)
		assert.Contains(t, rec.Body.String(), `"accrual":"500"`)
	})
}
