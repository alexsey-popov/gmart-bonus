// Пакет содержащий обработчик http запросов для сервиса программы лояльности
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/auth"
	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/alexsey-popov/gmart-bonus/internal/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

// Handler Обработчик запросов сервиса программы лояльности
type Handler struct {
	log       *slog.Logger
	rep       *repository.Repository
	validator *validator.Validator
}

// LoginRequest Структура данных для аутентификации пользователя
type LoginRequest struct {
	Login    string `json:"login" validate:"required,min=3,max=255" label:"Логин"`
	Password string `json:"password" validate:"required,min=3,max=255" label:"Пароль"`
}

// New Создание нового обработчика
func New(log *slog.Logger, rep *repository.Repository) Handler {

	v, err := validator.NewValidator()
	// Ошибка при создании валидатора не является критичной,
	// поэтому не прокидываем ошибку выше, а просто логируем её
	if err != nil {
		log.Error(err.Error(), slog.Any("error", err))
	}

	return Handler{
		log:       log,
		rep:       rep,
		validator: v,
	}
}

// GetRouter Обработчик запросов сервера
func (h Handler) GetRouter(cfg *config.Config) http.Handler {
	// Создаём роутер
	r := chi.NewRouter()

	// Логируем все запросы
	r.Use(httplog.RequestLogger(h.log, nil))

	// Обрабатываем сжатие для запросов и ответов с Content-Type application/json и text/html
	compressor := middleware.NewCompressor(5, "application/json", "text/html")
	r.Use(compressor.Handler)

	// Создаём объект аутентификации
	guard := auth.New(cfg.JwtToken)

	// Группа роутов только для неавторизированных пользователей
	r.Group(guard.GuestGroup(
		func(r chi.Router) {
			// Пропускаем только Content-Type application/json
			r.Use(middleware.AllowContentType("application/json"))

			// Регистрация
			r.Post("/register", h.Register(guard))

			// Аутентификация
			r.Post("/login", h.Login(guard))
		},
	))

	return r
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

		// Создаём нового пользователя
		userID, err := h.rep.CreateUser(req.Login, hashedPassword)
		if err != nil {
			// Если это ошибка уникальности - выдаём соответствующий код ответа
			if errors.Is(err, repository.ErrConflictUnique) {
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}

			// Если произошла другая ошибка - выдаём сухое сообщение без конкретики
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

		response, err := json.Marshal(map[string]interface{}{
			"id":    userID,
			"login": req.Login,
		})
		if err != nil {
			h.log.Error("Ошибка при конвертации ответа в json", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)

		if err != nil {
			h.log.Error("Ошибка записи ответа в ResponseWriter", slog.Any("error", err))
		}
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

		// Получаем данные пользователя по логину
		userID, storedPassword, err := h.rep.GetUserIdAndPassword(req.Login)
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

		response, err := json.Marshal(map[string]interface{}{
			"id":    userID,
			"login": req.Login,
		})
		if err != nil {
			h.log.Error("Ошибка при конвертации ответа в json", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			h.log.Error("Ошибка записи ответа в ResponseWriter", slog.Any("error", err))
		}
	}
}
