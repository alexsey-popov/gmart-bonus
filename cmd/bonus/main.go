// Сервис, отвечающий за программу лояльности Гофермарт.
// Регистрация и аутентификация пользователей, учёт бонусов, взаимодействие с системой расчёта
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/server"
	"github.com/golang-migrate/migrate/v4"

	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// main запуск сервиса программы лояльности
func main() {
	// Создаём логгер
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Парсим флаги(os.Args[0] пропускаем т.к. это имя исполняемого файла) и env
	cfg, err := config.Parse(os.Args[1:], os.LookupEnv)
	if err != nil {
		log.Error(err.Error())
		panic(err)
	}

	// Создаём объект взаимодействия с базой
	db, err := connectDB(cfg.DatabaseURI)
	if err != nil {
		log.Error(err.Error())
		panic(err)
	}
	defer db.Close()

	// Сервер программы лояльности
	s := server.NewServer(cfg, log, db)

	if err = s.ListenAndServe(); err != nil {
		log.Error(err.Error())
	}

}

// connectDB - Подключение в БД и выполнение миграций
func connectDB(serverDSN string) (*sqlx.DB, error) {
	// Создаём объект взаимодействия с базой
	db, err := sqlx.Connect("pgx", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("ошибка при подключении к БД: %w", err)
	}

	// Создаём драйвер для миграций
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return db, fmt.Errorf("ошибка при создании драйвера БД: %w", err)
	}

	//   Создаём объект миграции на основе файлов с миграциями и подключения
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"pgx", driver)
	if err != nil {
		return db, fmt.Errorf("ошибка при подготовке к миграций БД: %w", err)
	}

	// Проводим миграции
	err = m.Up()
	// Ошибку migrate.ErrNoChange пропускаем
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return db, fmt.Errorf("ошибка при запуске миграций БД: %w", err)
	}

	return db, nil
}
