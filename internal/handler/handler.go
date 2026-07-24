// Пакет содержащий обработчик http запросов для сервиса программы поляльности
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// Handler Обработчик запросов сервиса программы лояльности
type Handler struct {
	log       *slog.Logger
	db        *sqlx.DB
	validator *validator.Validate
}

// New Создание нового обработчика
func New(log *slog.Logger, db *sqlx.DB) Handler {
	return Handler{
		log:       log,
		db:        db,
		validator: validator.New(),
	}
}

// TranslateFields Валидация структуры с переводом полей
func (h Handler) Validate(s any, fields map[string]string) error {

	h.validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if translate, ok := fields[name]; ok {
			name = translate
		}

		return name
	})

	return h.validator.Struct(s)
}

// Register Регистрация нового пользователя
func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Структура запроса
	var req struct {
		Login    string `json:"login" validate:"required,min=3,max=255"`
		Password string `json:"password" validate:"required,min=3,max=255"`
	}

	// Парсим данные
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Переводы полей структуры
	fields := map[string]string{
		"login":    "Логин",
		"password": "Пароль",
	}

	// Валидируем данные
	if err := h.Validate(req, fields); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("ошибка при хешировании пароля", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Добавляем нового пользователя в БД
	query := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

	var userID string
	err = h.db.QueryRow(query, req.Login, string(hashedPassword)).Scan(&userID)
	if err != nil {
		// Если произошла ошибка уникальности по полю login - выводим соответствующую ошибку
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == "idx_users_login_unique" {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		h.log.Error("ошибка при создании нового пользователя в БД", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    userID,
		"login": req.Login,
	})
}
