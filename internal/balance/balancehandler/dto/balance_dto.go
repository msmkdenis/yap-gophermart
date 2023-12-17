package dto

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/msmkdenis/yap-gophermart/internal/balance/model"
)

type BalanceResponse struct {
	Current   decimal.Decimal `json:"current"`
	Withdrawn decimal.Decimal `json:"withdrawn"`
}

func MapToBalanceResponse(balance model.Balance) BalanceResponse {
	return BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}
}

type BalanceWithdrawRequest struct {
	OrderNumber string          `json:"order" validate:"required"`
	Amount      decimal.Decimal `json:"sum" validate:"required"`
}

type WithdrawalResponse struct {
	OrderNumber string          `json:"order"`
	Amount      decimal.Decimal `json:"sum"`
	ProcessedAt string          `json:"processed_at"`
}

func MapToWithdrawalResponse(withdrawal model.Withdrawal) WithdrawalResponse {
	return WithdrawalResponse{
		OrderNumber: withdrawal.OrderNumber,
		Amount:      withdrawal.Amount,
		ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
	}
}
