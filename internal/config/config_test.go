package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParse Тестирование парсинга конфига
func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		success bool
		want    *Config
	}{
		{
			name:    "positive - Пустой конфиг",
			args:    make([]string, 0),
			env:     make(map[string]string),
			success: true,
			want: &Config{
				UserAddress:    DefaultUserAddress,
				DatabaseURI:    DefaultDatabaseURI,
				AccrualAddress: DefaultAccrualAddress,
				JwtToken:       DefaultJwtToken,
			},
		},
		{
			name: "positive - Только env",
			args: make([]string, 0),
			env: map[string]string{
				EnvUserAddress:    "localhost:1111",
				EnvDatabaseURI:    "database",
				EnvAccrualAddress: "localhost:2222",
				EnvJwtToken:       "env token",
			},
			success: true,
			want: &Config{
				UserAddress:    "localhost:1111",
				DatabaseURI:    "database",
				AccrualAddress: "localhost:2222",
				JwtToken:       "env token",
			},
		},
		{
			name: "positive - Только флаги",
			args: []string{
				"-" + FlagUserAddress, "localhost:3333",
				"-" + FlagDatabaseURI, "different_database",
				"-" + FlagAccrualAddress, "localhost:4444",
				"-" + FlagJwtToken, "flag token",
			},
			env:     make(map[string]string),
			success: true,
			want: &Config{
				UserAddress:    "localhost:3333",
				DatabaseURI:    "different_database",
				AccrualAddress: "localhost:4444",
				JwtToken:       "flag token",
			},
		},
		{
			name: "positive - флаги поверх env",
			args: []string{
				"-" + FlagUserAddress, "localhost:3333",
				"-" + FlagDatabaseURI, "different_database",
				"-" + FlagAccrualAddress, "localhost:4444",
				"-" + FlagJwtToken, "flag token",
			},
			env: map[string]string{
				EnvUserAddress:    "localhost:1111",
				EnvDatabaseURI:    "database",
				EnvAccrualAddress: "localhost:2222",
				EnvJwtToken:       "env token",
			},
			success: true,
			want: &Config{
				UserAddress:    "localhost:3333",
				DatabaseURI:    "different_database",
				AccrualAddress: "localhost:4444",
				JwtToken:       "flag token",
			},
		},
		{
			name: "negative - Ошибка при парсинге флагов",
			args: []string{
				"-" + FlagUserAddress,
				"-" + FlagDatabaseURI,
				"-" + FlagAccrualAddress,
			},
			env:     make(map[string]string),
			success: false,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// lookupFunc заглушка вместо os.LookupEnv
			lookupFunc := func(name string) (string, bool) {
				value, ok := test.env[name]

				return value, ok
			}

			// Парсим конфиг
			cfg, err := Parse(test.args, lookupFunc)

			// Если тест должен завершиться успехом - проверяем отсутствие ошибки и сравниваем результаты
			// Иначе проверяем наличие ошибки
			if test.success {
				assert.NoError(t, err, "Ожидалось отсутствие ошибки, но вернулось %s", err)
				assert.Equal(t, test.want, cfg, "Несоответствие результатов ожидаемому значению")
			} else {
				assert.Error(t, err, "Ожидалась ошибка, но получили nil")
			}

		})
	}
}
