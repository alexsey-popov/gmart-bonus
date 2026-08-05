package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/shopspring/decimal"
)

// GetUserBalance Получение баланса пользователя
func (h Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем баланс пользователя
	balance, err := h.rep.GetUserBalance(r.Context(), userId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Создаём json ответ
	response, err := json.Marshal(balance)
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

// Реквест для списания бонусов
type CreateWithdrawalRequest struct {
	Order string          `validate:"required,number,order" label:"Номер заказа"`
	Sum   decimal.Decimal `validate:"required" label:"Сумма"`
}

// CreateWithdrawal Списать бонусы
func (h Handler) CreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	//Получаем данные запроса
	var request CreateWithdrawalRequest
	if err = json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.log.Error("ошибка при чтении тела запроса", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Валидируем данные запроса
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

		http.Error(w, h.validator.TransErrors(err).Error(), http.StatusBadRequest)
		return
	}

	// Проверяем, чтобы сумма бонусов была больше нуля
	if request.Sum.LessThanOrEqual(decimal.New(0, 0)) {
		http.Error(w, "Сумма бонусов к списанию должна быть больше нуля", http.StatusUnprocessableEntity)
		return
	}

	// Получаем баланс пользователя
	balance, err := h.rep.CreateWithdrawal(r.Context(), userId, request.Order, request.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientFunds) {
			http.Error(w, "Недостаточно бонусов на счёте", http.StatusPaymentRequired)
			return
		}

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Создаём json ответ
	response, err := json.Marshal(balance)
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

// GetUserWithdrawals История списаний бонусов
func (h Handler) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем список списаний
	withdrawals, err := h.rep.GetUserWithdrawals(r.Context(), userId)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Если список списаний пустой - возвращаем 204
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Подготавливаем данные к сериализации (убираем лишние поля, переименновываем некоторые)
	type responseItem struct {
		Order       string               `json:"order"`
		Sum         *decimal.NullDecimal `json:"sum"`
		ProcessedAt time.Time            `json:"processed_at"`
	}
	var responseItems []responseItem
	for _, item := range withdrawals {
		responseItems = append(responseItems, responseItem{
			Order:       item.OrderNumber,
			Sum:         item.Sum,
			ProcessedAt: item.ProcessedAt,
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
