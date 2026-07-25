// Сервер для сервиса программы лояльности
package server

import (
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/jmoiron/sqlx"
)

// Server Объект сервера программы лояльности
type Server struct {
	cfg *config.Config
	log *slog.Logger
	db  *sqlx.DB
}

// New Создание нового сервера
func NewServer(cfg *config.Config, log *slog.Logger, db *sqlx.DB) Server {
	return Server{
		cfg: cfg,
		log: log,
		db:  db,
	}
}

// ListenAndServe Запуск сервера
func (s Server) ListenAndServe() error {
	// Создаём роутер
	r := chi.NewRouter()

	// Логируем все запросы
	r.Use(httplog.RequestLogger(s.log, nil))

	// Обрабатываем сжатие для запросов и ответов с Content-Type application/json и text/html
	compressor := middleware.NewCompressor(5, "application/json", "text/html")
	r.Use(compressor.Handler)

	// Создаём объект аутентификации
	guard := auth.New(s.cfg.JwtToken)

	// Создаём обработчик запросов
	h := handler.New(s.log, s.db)

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

	s.log.Info("Запуск сервера программы лояльности по адресу: " + s.cfg.UserAddress)

	return http.ListenAndServe(s.cfg.UserAddress, r)
}
