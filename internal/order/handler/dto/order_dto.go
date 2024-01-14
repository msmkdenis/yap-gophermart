package dto

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"

	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderResponse struct {
	Number     string          `json:"number"`
	Status     string          `json:"status"`
	Accrual    decimal.Decimal `json:"accrual,omitempty"`
	UploadedAt string          `json:"uploaded_at"`
}

func MapToOrderResponse(order model.Order) OrderResponse {
	return OrderResponse{
		Number:     order.Number,
		Status:     order.Status,
		Accrual:    order.Accrual,
		UploadedAt: order.UploadedAt.Format(time.RFC3339),
	}
}

func (o *OrderResponse) MarshalJSON() ([]byte, error) {
	var jsonResponse []byte
	var err error
	if !o.Accrual.IsZero() {
		jsonResponse, err = json.Marshal(&struct {
			Number     string          `json:"number"`
			Status     string          `json:"status"`
			Accrual    decimal.Decimal `json:"accrual,omitempty"`
			UploadedAt string          `json:"uploaded_at"`
		}{
			Number:     o.Number,
			Status:     o.Status,
			Accrual:    o.Accrual,
			UploadedAt: o.UploadedAt,
		})
	} else {
		jsonResponse, err = json.Marshal(&struct {
			Number     string `json:"number"`
			Status     string `json:"status"`
			UploadedAt string `json:"uploaded_at"`
		}{
			Number:     o.Number,
			Status:     o.Status,
			UploadedAt: o.UploadedAt,
		})
	}
	return jsonResponse, err
}
