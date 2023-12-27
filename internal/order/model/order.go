package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	ID         string          `db:"id"`
	Number     string          `db:"number"`
	UserLogin  string          `db:"user_login"`
	UploadedAt time.Time       `db:"uploaded_at"`
	Accrual    decimal.Decimal `db:"http"`
	Status     string          `db:"status"`
}
