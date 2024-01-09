package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Balance struct {
	ID        string          `db:"id"`
	UserLogin string          `db:"user_login"`
	Current   decimal.Decimal `db:"current"`
	Withdrawn decimal.Decimal `db:"withdrawn"`
}

type Withdrawal struct {
	ID          string          `db:"id"`
	OrderNumber string          `db:"order_number"`
	UserLogin   string          `db:"user_login"`
	Amount      decimal.Decimal `db:"sum"`
	ProcessedAt time.Time       `db:"processed_at"`
}
