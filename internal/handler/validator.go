package handler

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/ru"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	ru_translations "github.com/go-playground/validator/v10/translations/ru"
)

// Validator Кастомный валидатор с ошибками на русском + подтягиванием полей из тега label
type Validator struct {
	validator  *validator.Validate
	translator ut.Translator
}

// Validate Валидация структуры с ошибками на русском языке
func (v Validator) Validate(item any) error {
	// Проводим валидацию
	if err := v.validator.Struct(item); err != nil {
		// Приводим ошибку к типу validator.ValidationErrors
		errs, ok := err.(validator.ValidationErrors)
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

	return nil
}

// NewValidator Создание нового валидатора
func NewValidator() (*Validator, error) {
	// Создаём валидатор
	validate := validator.New()

	// Указываем, что названия полей нужно брать из тега `label`
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("label")
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Настраиваем локализацию на русский язык
	ruLocale := ru.New()
	uni := ut.New(ruLocale, ruLocale)
	trans, _ := uni.GetTranslator("ru")

	v := &Validator{
		validator:  validate,
		translator: trans,
	}

	// Регистрируем встроенные русские переводы в валидаторе
	err := ru_translations.RegisterDefaultTranslations(v.validator, trans)
	if err != nil {
		return v, fmt.Errorf("ошибка регистрации переводов в валидаторе: %w", err)
	}

	return v, nil
}
