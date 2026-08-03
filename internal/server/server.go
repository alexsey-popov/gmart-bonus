// Сервер для сервиса программы лояльности
package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/handler"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/jmoiron/sqlx"
)

// Server Объект сервера программы лояльности
type Server struct {
	srv *http.Server
	cfg *config.Config
	log *slog.Logger
	rep *repository.Repository
}

// New Создание нового сервера
func NewServer(cfg *config.Config, log *slog.Logger, db *sqlx.DB) (Server, error) {
	// Создаём репозиторий
	rep := repository.NewRepository(db, log)

	// Создаём обработчик запросов
	h, err := handler.New(log, rep)
	if err != nil {
		return Server{}, err
	}

	return Server{
		srv: &http.Server{
			Addr:    cfg.UserAddress,
			Handler: h.GetRouter(cfg),
		},
		cfg: cfg,
		log: log,
		rep: rep,
	}, nil
}

// ListenAndServe Запуск сервера
func (s Server) ListenAndServe() error {
	s.log.Info("Запуск сервера программы лояльности по адресу: " + s.cfg.UserAddress)

	// Штатное завершение работы сервера не считаем ошибкой
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.log.Error("ошибка в работе сервера: ", slog.Any("error", err))
		return err
	}

	return nil
}

// Shutdown Завершение работы сервера
func (s Server) Shutdown(ctx context.Context) error {
	err := s.srv.Shutdown(ctx)
	if err != nil {
		s.log.Error("ошибка при остановке сервера", slog.Any("error", err))

		return err
	}

	return nil
}
