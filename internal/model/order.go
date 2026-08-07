package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus Статус заказа
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"        // Вознаграждение за заказ рассчитывается
	OrderStatusProcessing OrderStatus = "PROCESSING" // Вознаграждение за заказ рассчитывается
	OrderStatusInvalid    OrderStatus = "INVALID"    // Система расчёта вознаграждений отказала в расчёте
	OrderStatusProcessed  OrderStatus = "PROCESSED"  // Информация о расчёте успешно получена и обработана
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
