package dto

import (
	"time"

	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderResponse struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
}

func MapToOrderResponse(order model.Order) OrderResponse {
	return OrderResponse{
		Number:     order.Number,
		Status:     order.Status,
		UploadedAt: order.UploadedAt.Format(time.RFC3339),
	}
}
