// Пакет содержащий обработчик http запросов для сервиса программы лояльности
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/alexsey-popov/gmart-bonus/internal/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/jwtauth/v5"
	"github.com/shopspring/decimal"
)

// Repository Интерфейс репозитория, используемый обработчиком
type Repository interface {
	CreateUser(ctx context.Context, login, passwordHash string) (userID string, err error)
	GetUserIdAndPassword(ctx context.Context, login string) (userID, password string, err error)
	GetUserBalance(ctx context.Context, userId string) (balance model.Balance, err error)
	CreateWithdrawal(ctx context.Context, userId, order string, sum decimal.Decimal) (balance model.Balance, err error)
	GetUserWithdrawals(ctx context.Context, userId string) (withdrawals []model.Withdrawal, err error)
	CreateOrder(ctx context.Context, userId, number string) (model.Order, error)
	GetUserOrders(ctx context.Context, userId string) (orders []model.Order, err error)
}

// Handler Обработчик запросов сервиса программы лояльности
type Handler struct {
	log       *slog.Logger
	rep       Repository
	validator *validator.Validator
}

// New Создание нового обработчика
func New(log *slog.Logger, rep Repository) (*Handler, error) {
	// Создаём валидатор
	v, err := validator.NewValidator()
	if err != nil {
		log.Error(err.Error(), slog.Any("error", err))
		return &Handler{}, err
	}

	return &Handler{
		log:       log,
		rep:       rep,
		validator: v,
	}, nil
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

				// Получение баланса пользователя
				r.Get("/balance", h.GetUserBalance)

				// Списание бонусов
				r.Post("/balance/withdraw", h.CreateWithdrawal)

				// История списаний бонусов
				r.Get("/withdrawals", h.GetUserWithdrawals)
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
