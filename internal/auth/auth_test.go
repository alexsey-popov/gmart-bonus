package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew Тестирование создания объекта аутентификации
func TestNew(t *testing.T) {
	secret := "secret"
	guard := New(secret)
	assert.NotNil(t, guard)
	assert.NotNil(t, guard.jwt)
}

// TestNewUserToken Проверка создания токена аутентификации пользователя
func TestNewUserToken(t *testing.T) {
	secret := "secret"
	guard := New(secret)
	userId := "user123"

	tokenString, expiredAt, err := guard.NewUserToken(userId)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)
	assert.WithinDuration(t, time.Now().Add(TokenExpDuration), expiredAt, 10*time.Second)

	// Проверяем содержимое токена
	token, err := guard.jwt.Decode(tokenString)
	require.NoError(t, err)

	var val string
	err = token.Get("user_id", &val)
	require.NoError(t, err)
	assert.Equal(t, userId, val)
}

// TestGuestMiddleware Проверка посредника GuestMiddleware (только для неаутентифицированных пользователей)
func TestGuestMiddleware(t *testing.T) {
	secret := "secret"
	guard := New(secret)

	// Middleware должен работать в связке с Verifier
	h := jwtauth.Verifier(guard.jwt)(guard.GuestMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	t.Run("GuestMiddleware без токена", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GuestMiddleware с токеном пользователя токена", func(t *testing.T) {
		tokenString, _, _ := guard.NewUserToken("user123")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// TestAuthGroup Проверка группы роутов только для аутентифицированных пользователей
func TestAuthGroup(t *testing.T) {
	secret := "secret"
	guard := New(secret)
	userId := "user123"

	r := chi.NewRouter()
	r.Group(guard.AuthGroup(func(r chi.Router) {
		r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
			_, claims, _ := jwtauth.FromContext(r.Context())
			assert.Equal(t, userId, claims["user_id"])
			w.WriteHeader(http.StatusOK)
		})
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("unauthorized", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/protected")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("authorized", func(t *testing.T) {
		tokenString, _, _ := guard.NewUserToken(userId)
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestGuestGroup Проверка группы роутов только для неаутентифицированных пользователей
func TestGuestGroup(t *testing.T) {
	secret := "secret"
	guard := New(secret)

	r := chi.NewRouter()
	r.Group(guard.GuestGroup(func(r chi.Router) {
		r.Get("/guest-only", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("no_token", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/guest-only")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("with_token", func(t *testing.T) {
		tokenString, _, _ := guard.NewUserToken("user123")
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/guest-only", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}
