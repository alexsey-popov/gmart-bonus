package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Request struct {
	Name   string `validate:"required,min=3,max=255" label:"Логин"`
	Number string `validate:"order"`
}

// TestIsValidLuhn Проверка работы валидации по алгоритму Луна
func TestIsValidLuhn(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		success bool
	}{
		{
			name:    "positive - корректный номер",
			number:  "12345678903",
			success: true,
		},
		{
			name:    "negative - некорректный номер",
			number:  "123456789030",
			success: false,
		},
		{
			name:    "negative - недопустимые символы",
			number:  "A1234567A89030A",
			success: false,
		},
		{
			name:    "negative - пустая строка",
			number:  "",
			success: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			answer := IsValidLuhn(test.number)

			assert.Equal(t, test.success, answer, "Ожидался $v, а получили $v", test.success, answer)
		})
	}
}

// TestValidator_Validate Проверка функции Validate у структуры Validator
func TestValidator_Validate(t *testing.T) {
	v, errInit := NewValidator()
	require.NoError(t, errInit)

	tests := []struct {
		name    string
		req     Request
		wantErr bool
		errMsg  string
	}{
		{
			name: "positive - корректные данные",
			req: Request{
				Name:   "test",
				Number: "12345678903",
			},
			wantErr: false,
		},
		{
			name: "negative - проверка длины поля",
			req: Request{
				Name: "te",
			},
			wantErr: true,
		},
		{
			name: "negative - проверка обязательности поля",
			req: Request{
				Name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.req)
			if tt.wantErr {
				assert.Error(t, err, "Ожидалась ошибка, но получили nil")
			} else {
				assert.NoError(t, err, "Ожидалось отсутствие ошибки, но вернулось %s", err)
			}
		})
	}
}

// TestValidator_TransErrors Проверка перевода сообщений валидации
func TestValidator_TransErrors(t *testing.T) {
	v, errInit := NewValidator()
	require.NoError(t, errInit)

	tests := []struct {
		name      string
		customErr bool
		err       error
		req       Request
		success   bool
	}{
		{
			name:      "positive - Перевод ошибки",
			customErr: false,
			err:       nil,
			req: Request{
				Name: "te",
			},
			success: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Перевод ошибки
			var err, transErr error

			// Если ошибка кастомная - пытаемся перемести сразу её, иначе валидируем объект
			if test.customErr {
				err = test.err
				transErr = v.TransErrors(err)
			} else {
				err = v.Validate(test.req)
				assert.Error(t, err, "Ожидалась ошибка, но получили nil")

				transErr = v.TransErrors(err)
			}

			if test.success {
				assert.NotEqual(t, err.Error(), transErr.Error(), "Ожидалось, что сообщение изменится, но оно не изменилось: %s", err.Error())

			} else {
				assert.Equal(t, err.Error(), transErr.Error(), "Ожидалось, что сообщение не изменится, но оно изменилось с %s на %s", err.Error(), transErr.Error())
			}

		})
	}
}

// TestValidator_ErrorIs Проверка является ли ошибка валидации связанной с указанным полем и правилом
func TestValidator_ErrorIs(t *testing.T) {
	v, errInit := NewValidator()
	require.NoError(t, errInit)

	tests := []struct {
		name      string
		customErr bool
		err       error
		req       Request
		failed    bool
		success   bool
		Field     string
		Rule      string
	}{
		{
			name:      "positive - ошибка произошла в указанном поле по указанному правилу валидации",
			customErr: false,
			err:       nil,
			req: Request{
				Name: "te",
			},
			failed:  false,
			success: true,
			Field:   "Логин",
			Rule:    "min",
		},
		{
			name:      "negative - ошибка произошла в указанном поле, но не по указанному правилу валидации",
			customErr: false,
			err:       nil,
			req: Request{
				Name: "te",
			},
			failed:  false,
			success: false,
			Field:   "Логин",
			Rule:    "max",
		},
		{
			name:      "negative - ошибка произошла в другом поле по указанному правилу валидации",
			customErr: false,
			err:       nil,
			req: Request{
				Name: "te",
			},
			failed:  false,
			success: false,
			Field:   "Number",
			Rule:    "min",
		},
		{
			name:      "negative - ошибка не относится к ошибкам валидации",
			customErr: true,
			err:       errors.New("точно не ошибка валидации"),
			req:       Request{},
			failed:    true,
			success:   false,
			Field:     "Number",
			Rule:      "min",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error

			if test.customErr {
				err = test.err
			} else {
				err = v.Validate(test.req)
			}

			answer, err2 := v.ErrorIs(err, test.Field, test.Rule)

			if test.failed {
				assert.Error(t, err2, "Ожидалась ошибка, но получили nil")
			} else {
				assert.NoError(t, err2, "Ожидалось отсутствие ошибки, но вернулось %s", err2)
			}

			assert.Equal(t, test.success, answer, "Ожидался $v, а получили $v", test.success, answer)
		})
	}
}
