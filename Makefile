
# Название файла отчётом по покрытию кода тестами
COVER_FILE?=cover.out

# Запускаем все тесты
tests:
	go test ./... -count=1

# Подсчёт покрытия кода тестами с выводом в файл
tests-cover:
	go test ./... --coverprofile=$(COVER_FILE)

# Подсчёт покрытия кода тестами с просмотром в браузере
tests-cover-view: tests-cover
	go tool cover -html=$(COVER_FILE)

tests-cover-total: tests-cover
	go tool cover -func=$(COVER_FILE)


