package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegister Проверка обработчика регистрации на ранних ошибках (до обращения к репозиторию)
func TestRegister(t *testing.T) {
	guard := auth.New("secret")

	hash, err := guard.GetHash("password")
	require.NoError(t, err)

	rep := &fakeRepository{
		createUserFunc: func(ctx context.Context, login, passwordHash string) (string, error) {
			assert.Equal(t, "newuser", login)
			assert.NotEmpty(t, passwordHash)
			return "42", nil
		},
		getUserIdAndPasswordFunc: func(ctx context.Context, login string) (string, string, error) {
			return "42", hash, nil
		},
	}
	h := newTestHandler(t, rep)
	handlerFunc := h.Register(guard)

	t.Run("negative - некорректный JSON возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader("{invalid json"))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - ошибка валидации при пустых данных возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"","password":""}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - ошибка валидации при слишком коротком логине возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"ab","password":"password"}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("positive - успешная регистрация возвращает 200, тело и куку", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"newuser","password":"password"}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"id":"42","login":"newuser"}`, rec.Body.String())

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "jwt", cookies[0].Name)
		assert.NotEmpty(t, cookies[0].Value)
	})
}

// TestLogin Проверка обработчика аутентификации на ранних ошибках (до обращения к репозиторию)
func TestLogin(t *testing.T) {
	guard := auth.New("secret")

	hash, err := guard.GetHash("password")
	require.NoError(t, err)

	rep := &fakeRepository{
		createUserFunc: func(ctx context.Context, login, passwordHash string) (string, error) {
			assert.Equal(t, "newuser", login)
			assert.NotEmpty(t, passwordHash)
			return "42", nil
		},
		getUserIdAndPasswordFunc: func(ctx context.Context, login string) (string, string, error) {
			return "42", hash, nil
		},
	}
	h := newTestHandler(t, rep)
	handlerFunc := h.Login(guard)

	t.Run("negative - некорректный JSON возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader("{invalid json"))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - ошибка валидации при пустых данных возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"","password":""}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - ошибка валидации при слишком коротком пароле возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"login","password":"ab"}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("negative - неверный пароль возвращает 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"login","password":"wrongpassword"}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("positive - успешная аутентификация возвращает 200, тело и куку", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"login":"login","password":"password"}`))
		rec := httptest.NewRecorder()

		handlerFunc(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"id":"42","login":"login"}`, rec.Body.String())

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "jwt", cookies[0].Name)
		assert.NotEmpty(t, cookies[0].Value)
	})
}
