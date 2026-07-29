// Пакет содержащий обработчик http запросов для сервиса программы лояльности
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/alexsey-popov/gmart-bonus/internal/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/jwtauth/v5"
)

// Handler Обработчик запросов сервиса программы лояльности
type Handler struct {
	log       *slog.Logger
	rep       *repository.Repository
	validator *validator.Validator
}

// New Создание нового обработчика
func New(log *slog.Logger, rep *repository.Repository) Handler {

	v, err := validator.NewValidator()
	// Ошибка при создании валидатора не является критичной,
	// поэтому не прокидываем ошибку выше, а просто логируем её
	if err != nil {
		log.Error(err.Error(), slog.Any("error", err))
	}

	return Handler{
		log:       log,
		rep:       rep,
		validator: v,
	}
}

// GetRouter Обработчик запросов сервера
func (h Handler) GetRouter(cfg *config.Config) http.Handler {
	// Создаём роутер
	r := chi.NewRouter()

	// Логируем все запросы
	r.Use(httplog.RequestLogger(h.log, nil))

	// Обрабатываем сжатие для запросов и ответов с Content-Type application/json и text/html
	compressor := middleware.NewCompressor(5, "application/json", "text/html")
	r.Use(compressor.Handler)

	// Создаём объект аутентификации
	guard := auth.New(cfg.JwtToken)

	r.Route("/api/user", func(r chi.Router) {
		// Группа роутов только для неавторизированных пользователей
		r.Group(guard.GuestGroup(
			func(r chi.Router) {
				// Пропускаем только Content-Type application/json
				r.Use(middleware.AllowContentType("application/json"))

				// Регистрация
				r.Post("/register", h.Register(guard))

				// Аутентификация
				r.Post("/login", h.Login(guard))
			},
		))

		// Группа роутов только для авторизированных пользователей
		r.Group(guard.AuthGroup(
			func(r chi.Router) {
				// Создание заказа
				r.Post("/orders", h.CreateOrder)
				// Получение списка заказов
				r.Get("/orders", h.GetUserOrders)
			},
		))
	})

	return r
}

// GetUserId Получение Id пользователя
func (h Handler) GetUserId(r *http.Request) (string, error) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	userID, ok := claims["user_id"].(string)

	if !ok || userID == "" {
		return "", errors.New("пользователь не аутентифицирован")
	}

	return userID, nil
}
