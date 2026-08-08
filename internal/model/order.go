package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus Статус заказа
type OrderStatus string

const (
	// OrderStatusNew Вознаграждение за заказ рассчитывается
	OrderStatusNew OrderStatus = "NEW"

	// OrderStatusProcessing Вознаграждение за заказ рассчитывается
	OrderStatusProcessing OrderStatus = "PROCESSING"

	// OrderStatusInvalid Система расчёта вознаграждений отказала в расчёте
	OrderStatusInvalid OrderStatus = "INVALID"

	// OrderStatusProcessed Информация о расчёте успешно получена и обработана
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// Незаконченные статусы заказов
var UnfinishedOrderStatuses = []OrderStatus{OrderStatusNew, OrderStatusProcessing}

// Order Заказ
type Order struct {
	Id         string               `db:"id" json:"id"`
	UserId     string               `db:"user_id" json:"user_id"`
	Number     string               `db:"number" json:"number"`
	Status     OrderStatus          `db:"status" json:"status"`
	Accrual    *decimal.NullDecimal `db:"accrual" json:"accrual,omitempty"`
	UploadedAt time.Time            `db:"uploaded_at" json:"uploaded_at"`
}
