package model

import (
	"slices"

	"github.com/shopspring/decimal"
)

// AccrualStatus Статус начисления по заказу
type AccrualStatus string

const (
	// AccrualStatusRegistered Заказ зарегистрирован, но вознаграждение не рассчитано
	AccrualStatusRegistered AccrualStatus = "REGISTERED"

	// AccrualStatusProcessing Расчёт начисления в процессе
	AccrualStatusProcessing AccrualStatus = "PROCESSING"

	// AccrualStatusProcessed Расчёт начисления окончен
	AccrualStatusProcessed AccrualStatus = "PROCESSED"

	// AccrualStatusInvalid Заказ не принят к расчёту, и вознаграждение не будет начислено
	AccrualStatusInvalid AccrualStatus = "INVALID"
)

// GetOrderStatus Получаем статус заказа в зависимости от статуса начисления
// Если что-то пошло не так - выставляем статус INVALID
func (accrualStatus AccrualStatus) GetOrderStatus() OrderStatus {
	orderStatus := OrderStatusInvalid

	switch accrualStatus {
	case AccrualStatusRegistered, AccrualStatusProcessing:
		orderStatus = OrderStatusProcessing
	case AccrualStatusInvalid:
		orderStatus = OrderStatusInvalid
	case AccrualStatusProcessed:
		orderStatus = OrderStatusProcessed
	}

	return orderStatus
}

// Окончательные статусы начисления по заказу
var FinalAccrualStatuses = []AccrualStatus{AccrualStatusProcessed, AccrualStatusInvalid}

// isFinal Является ли статус начисления окончательным
func (accrualStatus AccrualStatus) IsFinal() bool {
	return slices.Contains(FinalAccrualStatuses, accrualStatus)
}

// AccrualOrder Начисление по заказу
type AccrualOrder struct {
	Order   int                  `json:"order"`
	Status  AccrualStatus        `json:"status"`
	Accrual *decimal.NullDecimal `json:"accrual,omitempty"`
}
