package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/alexsey-popov/gmart-bonus/internal/repository"
)

type CreateOrderRequest struct {
	Order string `validate:"required,number,order" label:"Номер заказа"`
}

// CreateOrder Создание заказа
func (h Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Читаем содержимое запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("ошибка при чтении тела запроса", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Создаём и валидируем данные
	request := CreateOrderRequest{
		Order: string(body),
	}
	if err = h.validator.Validate(request); err != nil {
		// Проверяем произошла ли ошибка на этапе проверки корректности номера заказа
		failedLuthn, err2 := h.validator.ErrorIs(err, "Номер заказа", "order")
		if err2 != nil {
			h.log.Error("валидатор не смог проверить ошибку на соответствие поля и тега", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Если ошибка связана с проверкой алгоритмом Луна - выводи 422 статус
		if failedLuthn {
			http.Error(w, "Некорректный номер заказа", http.StatusUnprocessableEntity)
			return
		}

		// Прочие ошибки возвращаем пользователю
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Пытаемся создать заказ
	order, err := h.rep.CreateOrder(r.Context(), userId, request.Order)
	if err != nil {
		// Смотрим является ли ошибка конфликтом уникальности по полю
		isConflictUnique := errors.Is(err, repository.ErrConflictUnique)

		// Если это конфликт уникальности и user_id совпадает - выдаём статус 200
		if isConflictUnique && order.UserId == userId {
			http.Error(w, "Номер заказа уже был загружен этим пользователем", http.StatusOK)
			return
		}

		// Если это конфликт уникальности и user_id не совпадает - выдаём статус 409
		if isConflictUnique && order.UserId != userId {
			http.Error(w, "Номер заказа уже был загружен другим пользователем", http.StatusConflict)
			return
		}

		// В любом другом случае отдаём 500
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Создаём json ответ
	response, err := json.Marshal(order)
	if err != nil {
		h.log.Error("ошибка при сериализации json", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, err = w.Write(response)
	if err != nil {
		h.log.Error("Ошибка записи ответа в ResponseWriter", slog.Any("error", err))
	}
}

// GetUserOrders Получение списка заказов пользователя
func (h Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем список заказов
	orders, err := h.rep.GetUserOrders(r.Context(), userId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Если список заказов пустой - возвращаем 204
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Создаём json ответ
	response, err := json.Marshal(orders)
	if err != nil {
		h.log.Error("ошибка при сериализации json", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(response)
	if err != nil {
		h.log.Error("Ошибка записи ответа в ResponseWriter", slog.Any("error", err))
	}
}
