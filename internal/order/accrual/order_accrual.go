package accrual

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/go-resty/resty/v2"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderAccrual struct {
	*resty.Client
	logger *zap.Logger
}

func NewOrderAccrual(accrualEndpoint string, logger *zap.Logger) *OrderAccrual {
	orderAccrual := &OrderAccrual{resty.New(), logger}
	orderAccrual.SetBaseURL(accrualEndpoint)

	return orderAccrual
}

func (o *OrderAccrual) QueryUpdateOrder(orderNumber string) (*model.Order, error) {
	var order model.Order
	r, err := o.R().SetResult(&order).Get("/api/orders/" + orderNumber)
	if err != nil {
		o.logger.Error("error while processing accrual", zap.Error(err))
		return nil, err
	}

	if r.StatusCode() == http.StatusNoContent {
		o.logger.Info("error while processing accrual", zap.Error(err))
		return nil, apperrors.ErrOrderNotFound
	}

	if r.StatusCode() == http.StatusTooManyRequests {
		o.logger.Error("error while processing accrual", zap.Error(err))
		return nil, apperrors.ErrRateLimit
	}

	return &order, nil
}
