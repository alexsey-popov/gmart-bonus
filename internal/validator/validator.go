package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/ru"
	ut "github.com/go-playground/universal-translator"
	val "github.com/go-playground/validator/v10"
	ru_translations "github.com/go-playground/validator/v10/translations/ru"
)

// Validator Кастомный валидатор с ошибками на русском + подтягиванием полей из тега label
type Validator struct {
	validator  *val.Validate
	translator ut.Translator
}

// Validate Валидация структуры с ошибками на русском языке
func (v Validator) Validate(item any) error {
	return v.validator.Struct(item)
}

// Перевод ошибок
func (v Validator) TransErrors(err error) error {
	// Приводим ошибку к типу validator.ValidationErrors
	errs, ok := err.(val.ValidationErrors)
	if !ok {
		return err
	}

	// Собираем ошибки на русском языке
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.Translate(v.translator))
	}
	return fmt.Errorf("%s", strings.Join(messages, ", "))
}

// ErrorIs Проверка ошибки на предмет конкретного поля и правила валидации
func (v Validator) ErrorIs(err error, field, tag string) (bool, error) {
	// Приводим ошибку к типу validator.ValidationErrors и пытаемся найти совпадение с полем и правилом
	if errs, ok := err.(val.ValidationErrors); ok {
		for _, e := range errs {
			// Проверяем имя поля и сработавшее правило (тег)
			if e.Field() == field && e.Tag() == tag {
				return true, nil
			}
		}

		return false, nil
	}

	return false, errors.New("переданная ошибка не соответствует типу ValidationErrors")
}

// NewValidator Создание нового валидатора
func NewValidator() (*Validator, error) {
	// Создаём валидатор
	validate := val.New()

	// Указываем, что названия полей нужно брать из тега `label`
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("label")
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Регистрируем правило для проверки строки из чисел по алгоритму Луна
	err := validate.RegisterValidation("order", func(fl val.FieldLevel) bool {
		return IsValidLuhn(fl.Field().String())
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка регистрации правила order: %w", err)
	}

	// Настраиваем локализацию на русский язык
	ruLocale := ru.New()
	uni := ut.New(ruLocale, ruLocale)
	trans, _ := uni.GetTranslator("ru")

	v := &Validator{
		validator:  validate,
		translator: trans,
	}

	// Регистрируем встроенные русские переводы в валидаторе
	err = ru_translations.RegisterDefaultTranslations(v.validator, trans)
	if err != nil {
		return v, fmt.Errorf("ошибка регистрации переводов в валидаторе: %w", err)
	}

	return v, nil
}

// IsValidLuhn проверяет корректность строки по алгоритму Луна
func IsValidLuhn(number string) bool {
	sum := 0
	second := false

	// Пустую строку считаем некорректным номером
	if len(number) == 0 {
		return false
	}

	// Идем справа налево
	for i := len(number) - 1; i >= 0; i-- {
		char := number[i]

		// Если среди символов есть что-то отличное от чисел - выдаём false
		if char < '0' || char > '9' {
			return false
		}

		val := int(char - '0')

		if second {
			val *= 2
			if val > 9 {
				val -= 9
			}
		}

		sum += val
		second = !second
	}

	return sum%10 == 0
}
