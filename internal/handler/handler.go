// Пакет содержащий обработчик http запросов для сервиса программы лояльности
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// Handler Обработчик запросов сервиса программы лояльности
type Handler struct {
	log       *slog.Logger
	db        *sqlx.DB
	validator *Validator
}

// LoginRequest Структура данных для аутентификации пользователя
type LoginRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=255" label:"Логин"`
	Password string `json:"password" validate:"required,min=3,max=255" label:"Пароль"`
}

// New Создание нового обработчика
func New(log *slog.Logger, db *sqlx.DB) Handler {

	v, err := NewValidator()
	// Ошибка при создании валидатора не является критичной,
	// поэтому не прокидываем ошибку выше, а просто логируем её
	if err != nil {
		log.Error(err.Error(), slog.Any("error", err))
	}

	return Handler{
		log:       log,
		db:        db,
		validator: v,
	}
}

// Register Регистрация нового пользователя
func (h Handler) Register(guard *auth.Guard) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Создаём переменную для данных запроса
		var req LoginRequest

		// Парсим данные
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Валидируем данные
		if err := h.validator.Validate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Хешируем пароль
		hashedPassword, err := guard.GetHash(req.Password)
		if err != nil {
			h.log.Error(err.Error(), slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Добавляем нового пользователя в БД
		query := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

		var userID string
		err = h.db.QueryRow(query, req.Login, hashedPassword).Scan(&userID)
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

		// Создаём новый токен аутентификации
		token, expiredAt, err := guard.NewUserToken(userID)
		if err != nil {
			h.log.Error(err.Error(), slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Записываем куку в ответ
		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  expiredAt,
			Path:     "/",                     // Действие куки распространяется с корня сайта
			HttpOnly: true,                    // закрывает доступ к куке из JavaScript
			SameSite: http.SameSiteStrictMode, // защита от CSRF атак
		})

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    userID,
			"login": req.Login,
		})
	}
}

// Login Аутентификация пользователя
func (h Handler) Login(guard *auth.Guard) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Создаём переменную для данных запроса
		var req LoginRequest

		// Парсим данные
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Валидируем данные
		if err := h.validator.Validate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Проверяем наличие пользователя в БД
		query := `SELECT id, password FROM users WHERE login = $1`

		var userID, storedPassword string
		err := h.db.QueryRow(query, req.Login).Scan(&userID, &storedPassword)
		if err != nil {
			http.Error(w, "Некорректный логин или пароль", http.StatusUnauthorized)
			return
		}

		// Сверяем пароль с хешем из БД
		if !guard.CheckPassword(req.Password, storedPassword) {
			http.Error(w, "Некорректный логин или пароль", http.StatusUnauthorized)
			return
		}

		// Создаём новый токен аутентификации
		token, expiredAt, err := guard.NewUserToken(userID)
		if err != nil {
			h.log.Error(err.Error(), slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Записываем куку в ответ
		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  expiredAt,
			Path:     "/",                     // Действие куки распространяется с корня сайта
			HttpOnly: true,                    // закрывает доступ к куке из JavaScript
			SameSite: http.SameSiteStrictMode, // защита от CSRF атак
		})

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    userID,
			"login": req.Login,
		})
	}
}
