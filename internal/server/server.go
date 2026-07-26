// Сервер для сервиса программы лояльности
package server

import (
	"context"
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
	srv *http.Server
	cfg *config.Config
	log *slog.Logger
	db  *sqlx.DB
}

// New Создание нового сервера
func NewServer(cfg *config.Config, log *slog.Logger, db *sqlx.DB) Server {
	return Server{
		srv: &http.Server{
			Addr:    cfg.UserAddress,
			Handler: NewMux(cfg, log, db),
		},
		cfg: cfg,
		log: log,
		db:  db,
	}
}

// ListenAndServe Запуск сервера
func (s Server) ListenAndServe() error {
	s.log.Info("Запуск сервера программы лояльности по адресу: " + s.cfg.UserAddress)

	if err := s.srv.ListenAndServe(); err != nil {
		s.log.Error("ошибка в работе сервера: ", slog.Any("error", err))
		return err
	}

	return nil
}

// NewMux Обработчик запросов сервера
func NewMux(cfg *config.Config, log *slog.Logger, db *sqlx.DB) http.Handler {
	// Создаём роутер
	r := chi.NewRouter()

	// Логируем все запросы
	r.Use(httplog.RequestLogger(log, nil))

	// Обрабатываем сжатие для запросов и ответов с Content-Type application/json и text/html
	compressor := middleware.NewCompressor(5, "application/json", "text/html")
	r.Use(compressor.Handler)

	// Создаём объект аутентификации
	guard := auth.New(cfg.JwtToken)

	// Создаём обработчик запросов
	h := handler.New(log, db)

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

	return r
}

// Shutdown Завершение работы сервера
func (s Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
