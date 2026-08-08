package model

import (
	"github.com/shopspring/decimal"
)

// AccrualOrderStatus Статус начисления по заказу
type AccrualOrderStatus string

const (
	// AccrualOrderStatusRegistered Заказ зарегистрирован, но вознаграждение не рассчитано
	AccrualOrderStatusRegistered AccrualOrderStatus = "REGISTERED"

	// AccrualOrderStatusProcessing Расчёт начисления в процессе
	AccrualOrderStatusProcessing AccrualOrderStatus = "PROCESSING"

	// AccrualOrderStatusProcessed Расчёт начисления окончен
	AccrualOrderStatusProcessed AccrualOrderStatus = "PROCESSED"

	// AccrualOrderStatusInvalid Заказ не принят к расчёту, и вознаграждение не будет начислено
	AccrualOrderStatusInvalid AccrualOrderStatus = "INVALID"
)

// Незаконченные статусы начисления по заказу
var UnfinishedAccrualOrderStatuses = []AccrualOrderStatus{AccrualOrderStatusRegistered, AccrualOrderStatusProcessing}

// AccrualOrder Начисление по заказу
type AccrualOrder struct {
	Order   int                  `json:"order"`
	Status  AccrualOrderStatus   `json:"status"`
	Accrual *decimal.NullDecimal `json:"accrual,omitempty"`
}

// GetOrderStatus Получаем статус заказа в зависимости от статуса начисления
// Если что-то пошло не так - выставляем статус INVALID
func (ao AccrualOrder) GetOrderStatus() OrderStatus {
	switch ao.Status {
	case AccrualOrderStatusRegistered, AccrualOrderStatusProcessing:
		return OrderStatusProcessing
	case AccrualOrderStatusInvalid:
		return OrderStatusInvalid
	case AccrualOrderStatusProcessed:
		return OrderStatusProcessed
	}

	return OrderStatusInvalid
}
