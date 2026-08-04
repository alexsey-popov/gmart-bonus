package model

import "github.com/shopspring/decimal"

// Balance Баланс пользователя
type Balance struct {
	Current   decimal.Decimal `db:"current" json:"current"`
	Withdrawn decimal.Decimal `db:"withdrawn" json:"withdrawn"`
}
