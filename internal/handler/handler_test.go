package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/go-chi/jwtauth/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHandler Создание обработчика для тестов с подменённым репозиторием
func newTestHandler(t *testing.T, rep Repository) Handler {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := New(log, rep)
	require.NoError(t, err)

	return h
}

// fakeRepository Фейковая реализация Repository для тестов обработчиков
type fakeRepository struct {
	createUserFunc           func(ctx context.Context, login, passwordHash string) (string, error)
	getUserIdAndPasswordFunc func(ctx context.Context, login string) (string, string, error)
	getUserBalanceFunc       func(ctx context.Context, userId string) (model.Balance, error)
	createWithdrawalFunc     func(ctx context.Context, userId, order string, sum decimal.Decimal) (model.Balance, error)
	getUserWithdrawalsFunc   func(ctx context.Context, userId string) ([]model.Withdrawal, error)
	createOrderFunc          func(ctx context.Context, userId, number string) (model.Order, error)
	getUserOrdersFunc        func(ctx context.Context, userId string) ([]model.Order, error)
}

func (f *fakeRepository) CreateUser(ctx context.Context, login, passwordHash string) (string, error) {
	return f.createUserFunc(ctx, login, passwordHash)
}

func (f *fakeRepository) GetUserIdAndPassword(ctx context.Context, login string) (string, string, error) {
	return f.getUserIdAndPasswordFunc(ctx, login)
}

func (f *fakeRepository) GetUserBalance(ctx context.Context, userId string) (model.Balance, error) {
	return f.getUserBalanceFunc(ctx, userId)
}

func (f *fakeRepository) CreateWithdrawal(ctx context.Context, userId, order string, sum decimal.Decimal) (model.Balance, error) {
	return f.createWithdrawalFunc(ctx, userId, order, sum)
}

func (f *fakeRepository) GetUserWithdrawals(ctx context.Context, userId string) ([]model.Withdrawal, error) {
	return f.getUserWithdrawalsFunc(ctx, userId)
}

func (f *fakeRepository) CreateOrder(ctx context.Context, userId, number string) (model.Order, error) {
	return f.createOrderFunc(ctx, userId, number)
}

func (f *fakeRepository) GetUserOrders(ctx context.Context, userId string) ([]model.Order, error) {
	return f.getUserOrdersFunc(ctx, userId)
}

// requestWithUserID Создание запроса с проброшенным в контекст user_id
func requestWithUserID(t *testing.T, userID string) *http.Request {
	t.Helper()

	ja := jwtauth.New("HS256", []byte("secret"), nil)
	_, tokenString, err := ja.Encode(map[string]interface{}{"user_id": userID})
	require.NoError(t, err)

	token, err := ja.Decode(tokenString)
	require.NoError(t, err)

	ctx := jwtauth.NewContext(context.Background(), token, nil)
	return httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
}

// TestNew Проверка создания обработчика
func TestNew(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	h, err := New(log, nil)
	require.NoError(t, err)
	assert.NotNil(t, h.log)
	assert.NotNil(t, h.validator)
	assert.Nil(t, h.rep)
}

// TestGetRouter Проверка создания роутера и регистрации роутов
func TestGetRouter(t *testing.T) {
	h := newTestHandler(t, nil)
	cfg := &config.Config{JwtToken: "secret"}

	router := h.GetRouter(cfg)
	require.NotNil(t, router)

	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("negative - защищённый роут без токена возвращает 401", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/user/orders")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("negative - гостевой роут с неверным Content-Type возвращает 415", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/user/register", "text/plain", strings.NewReader("{}"))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("negative - несуществующий роут возвращает 404", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/user/unknown")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestGetUserId Проверка извлечения id пользователя из токена
func TestGetUserId(t *testing.T) {
	h := newTestHandler(t, nil)

	t.Run("positive - валидный user_id в контексте", func(t *testing.T) {
		req := requestWithUserID(t, "user123")

		userID, err := h.GetUserId(req)
		require.NoError(t, err)
		assert.Equal(t, "user123", userID)
	})

	t.Run("negative - пустой user_id", func(t *testing.T) {
		req := requestWithUserID(t, "")

		userID, err := h.GetUserId(req)
		require.Error(t, err)
		assert.Empty(t, userID)
	})

	t.Run("negative - токен отсутствует в контексте", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		userID, err := h.GetUserId(req)
		require.Error(t, err)
		assert.Empty(t, userID)
	})
}
