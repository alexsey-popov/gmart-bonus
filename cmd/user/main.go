// Сервис, отвечающий за программу лояльности Гофермарт.
// Регистрация и аутентификация пользователей, учёт бонусов, взаимодействие с системой расчёта
package main

import (
	"fmt"
	"os"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
)

// main запуск сервиса программы лояльности
func main() {
	// Парсим флаги(os.Args[0] пропускаем т.к. это имя исполняемого файла) и env
	c, err := config.Parse(os.Args[1:], os.LookupEnv)
	if err != nil {
		panic(err)
	}

	fmt.Println(c.UserAddress, c.DatabaseURI, c.AccrualAddress)
}
