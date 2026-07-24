// Пакет для реализации аутентификации
package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

// TokenExpDuration Длительность жизни токена
const TokenExpDuration = 7 * 24 * time.Hour

// Guard Объект для реализации аутентификации
type Guard struct {
	jwt *jwtauth.JWTAuth
}

// New Создание нового объекта аутентификации
func New(secret string) *Guard {
	return &Guard{
		jwt: jwtauth.New("HS256", []byte(secret), nil),
	}
}

// NewUserToken Создание нового токена пользователя
func (g Guard) NewUserToken(userId string) (string, time.Time, error) {
	// Данные пользователя (Claims)
	claims := map[string]interface{}{
		"user_id": userId,
	}

	// Устанавливаем время жизни токена
	jwtauth.SetExpiry(claims, time.Now().Add(TokenExpDuration))

	// Генерируем и подписываем токен
	token, tokenString, err := g.jwt.Encode(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("ошибка при создании токена пользователя: %w", err)
	}

	// Момент окончания действия токена
	expiredAt, _ := token.Expiration()

	return tokenString, expiredAt, nil
}

// GuestMiddleware Доступ только для неаутентифицированных пользователей
func (g Guard) GuestMiddleware(h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем токен из контекста запроса
		_, claims, err := jwtauth.FromContext(r.Context())

		// Если ошибки нет и claims существуют - значит пользователь уже авторизован и нужно вывести ошибку
		if err == nil && claims != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// GuestGroup Группа с доступом только для аутентифицированных пользователей
func (g Guard) GuestGroup(routes func(r chi.Router)) func(r chi.Router) {
	return func(r chi.Router) {
		// Ищем токен пользователя и пробрасываем его в контекст
		r.Use(jwtauth.Verifier(g.jwt))

		// Доступ только для неаутентифицированных пользователей
		r.Use(g.GuestMiddleware)

		r.Group(routes)
	}
}

// AuthMiddleware Группа с доступом только для аутентифицированных пользователей
func (g Guard) AuthGroup(routes func(r chi.Router)) func(r chi.Router) {
	return func(r chi.Router) {
		// Ищем токен пользователя и пробрасываем его в контекст
		r.Use(jwtauth.Verifier(g.jwt))

		// Доступ только для аутентифицированных пользователей
		r.Use(jwtauth.Authenticator(g.jwt))

		r.Group(routes)
	}
}
