// Конфигурация сервисов.
package config

import (
	"flag"
	"fmt"
)

const (
	// Адрес системы программы лояльности
	FlagUserAddress    = "a"
	EnvUserAddress     = "RUN_ADDRESS"
	DefaultUserAddress = "localhost:8080"

	// Адрес подключения к базе данных
	FlagDatabaseURI    = "d"
	EnvDatabaseURI     = "DATABASE_URI"
	DefaultDatabaseURI = ""

	// Адрес системы расчёта начислений
	FlagAccrualAddress    = "r"
	EnvAccrualAddress     = "ACCRUAL_SYSTEM_ADDRESS"
	DefaultAccrualAddress = "localhost:8081"
)

// Config - Структура конфигурации сервера
type Config struct {
	UserAddress    string
	DatabaseURI    string
	AccrualAddress string
}

// EnvLookupFunc заглушка для `os.LookupEnv`
type EnvLookupFunc func(string) (string, bool)

// getEnvString Получение строковых данных из env или выставление значения по умолчанию
func getEnvString(lookupFunc EnvLookupFunc, name string, defaultValue string) string {
	if value, ok := lookupFunc(name); ok {
		return value
	}

	return defaultValue
}

// Parse парсинг флагов и env (приоритет по убыванию: флаги, env, default)
func Parse(args []string, lookupFunc EnvLookupFunc) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("app", flag.ContinueOnError)

	// Получаем данные из env или их дефолтные значения
	userAddress := getEnvString(lookupFunc, EnvUserAddress, DefaultUserAddress)
	databaseURI := getEnvString(lookupFunc, EnvDatabaseURI, DefaultDatabaseURI)
	accrualAddress := getEnvString(lookupFunc, EnvAccrualAddress, DefaultAccrualAddress)

	// Получаем данные из флагов
	fs.StringVar(&cfg.UserAddress, FlagUserAddress, userAddress, "Адрес системы программы лояльности")
	fs.StringVar(&cfg.DatabaseURI, FlagDatabaseURI, databaseURI, "Адрес подключения к базе данных")
	fs.StringVar(&cfg.AccrualAddress, FlagAccrualAddress, accrualAddress, "Адрес системы расчёта начислений")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("ошибка при парсинге флагов: %w", err)
	}

	return cfg, nil
}
