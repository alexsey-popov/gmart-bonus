package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Order Заказ
type Order struct {
	Id         string               `db:"id" json:"id"`
	UserId     string               `db:"user_id" json:"user_id"`
	Number     string               `db:"number" json:"number"`
	Status     string               `db:"status" json:"status"`
	Accrual    *decimal.NullDecimal `db:"accrual" json:"accrual,omitempty"`
	UploadedAt time.Time            `db:"uploaded_at" json:"uploaded_at"`
}
