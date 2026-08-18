package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Withdrawal Списание бонусов
type Withdrawal struct {
	Id          string               `db:"id" json:"id"`
	UserId      string               `db:"user_id" json:"user_id"`
	OrderNumber string               `db:"order_number" json:"order_number"`
	Sum         *decimal.NullDecimal `db:"sum" json:"sum"`
	ProcessedAt time.Time            `db:"processed_at" json:"processed_at"`
}
