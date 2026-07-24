// Пакет для реализации аутентификации
package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jmoiron/sqlx"
)

// Guard Объект для реализации аутентификации
type Guard struct {
	token *jwtauth.JWTAuth
	db    *sqlx.DB
}

// New Создание нового объекта аутентификации
func New(secret string, db *sqlx.DB) Guard {
	return Guard{
		token: jwtauth.New("HS256", []byte(secret), nil),
		db:    db,
	}
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
		r.Use(jwtauth.Verifier(g.token))

		// Доступ только для неаутентифицированных пользователей
		r.Use(g.GuestMiddleware)

		r.Group(routes)
	}
}

// AuthMiddleware Группа с доступом только для аутентифицированных пользователей
func (g Guard) AuthGroup(routes func(r chi.Router)) func(r chi.Router) {
	return func(r chi.Router) {
		// Ищем токен пользователя и пробрасываем его в контекст
		r.Use(jwtauth.Verifier(g.token))

		// Доступ только для аутентифицированных пользователей
		r.Use(jwtauth.Authenticator(g.token))

		r.Group(routes)
	}
}
