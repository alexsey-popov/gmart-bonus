// Сервис, отвечающий за программу лояльности Гофермарт.
// Регистрация и аутентификация пользователей, учёт бонусов, взаимодействие с системой расчёта
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/server"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

// main запуск сервиса программы лояльности
func main() {
	// Создаём логгер
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Парсим флаги(os.Args[0] пропускаем т.к. это имя исполняемого файла) и env
	cfg, err := config.Parse(os.Args[1:], os.LookupEnv)
	if err != nil {
		log.Error(err.Error(),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	// Создаём объект взаимодействия с базой
	db, err := sqlx.Connect("pgx", cfg.DatabaseURI)
	if err != nil {
		log.Error("ошибка при подключении к БД",
			slog.Any("error", err),
		)
		os.Exit(1)
	}
	defer db.Close()

	// Выполняем миграции БД
	if err = migrateUp(db); err != nil {
		log.Error(err.Error(),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	// Создаём сервер программы лояльности
	s, err := server.NewServer(cfg, log, db)
	if err != nil {
		log.Error("ошибка при подключении к БД",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	// Контекст, который отменится при получении SIGINT или SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Создаём errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// Запускаем сервер в отдельной горутине
	g.Go(s.ListenAndServe)

	// Graceful shutdown HTTP-сервера
	g.Go(func() error {
		<-gCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.Shutdown(shutdownCtx)
	})

	// Ожидаем завершения всех горутин
	if err := g.Wait(); err != nil {
		log.Info("завершение с ошибкой", slog.Any("error", err))
	}

	log.Info("сервер остановлен")
}

// migrateUp Выполнение миграций БД
func migrateUp(db *sqlx.DB) error {
	// Создаём драйвер для миграций
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("ошибка при создании драйвера БД: %w", err)
	}

	//   Создаём объект миграции на основе файлов с миграциями и подключения
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"pgx", driver)
	if err != nil {
		return fmt.Errorf("ошибка при подготовке к миграций БД: %w", err)
	}

	// Проводим миграции
	err = m.Up()
	// Ошибку migrate.ErrNoChange пропускаем
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("ошибка при запуске миграций БД: %w", err)
	}

	return nil
}
