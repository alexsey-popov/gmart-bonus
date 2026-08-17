package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/shopspring/decimal"
)

// Реквест для связи заказа с пользователем
type CreateOrderRequest struct {
	Order string `validate:"required,number,order" label:"Номер заказа"`
}

// CreateOrder Связь заказа с пользователем
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
		http.Error(w, h.validator.TransErrors(err).Error(), http.StatusBadRequest)
		return
	}

	// Пытаемся создать заказ
	order, err := h.rep.CreateOrder(r.Context(), userId, request.Order)
	if err != nil {

		// Если произошла ошибка конфликтом уникальности по полю -
		// выводим ответ в зависимости от значения order.UserId
		if errors.Is(err, repository.ErrConflictUnique) {
			switch order.UserId {
			case userId:
				http.Error(w, "Номер заказа уже был загружен этим пользователем", http.StatusOK)
			case "":
				http.Error(w, "Номер заказа уже был загружен неизвестным пользователем", http.StatusConflict)
			default:
				http.Error(w, "Номер заказа уже был загружен другим пользователем", http.StatusConflict)
			}
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

	// Подготавливаем данные к сериализации (убираем лишние поля, переименновываем некоторые)
	type responseItem struct {
		Number     string               `json:"number"`
		Status     model.OrderStatus    `json:"status"`
		Accrual    *decimal.NullDecimal `json:"accrual,omitempty"`
		UploadedAt time.Time            `json:"uploaded_at"`
	}
	var responseItems []responseItem
	for _, item := range orders {
		responseItems = append(responseItems, responseItem{
			Number:     item.Number,
			Status:     item.Status,
			Accrual:    item.Accrual,
			UploadedAt: item.UploadedAt,
		})
	}

	// Создаём json ответ
	response, err := json.Marshal(responseItems)
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
