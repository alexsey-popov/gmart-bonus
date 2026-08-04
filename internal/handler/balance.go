package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Balance Получение баланса пользователя
func (h Handler) Balance(w http.ResponseWriter, r *http.Request) {
	// Получаем id пользователя
	userId, err := h.GetUserId(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем список заказов
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
