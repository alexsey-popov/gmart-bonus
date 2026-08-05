// migrations Пакет, который помогает зашить миграции в бинарник сервера
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
