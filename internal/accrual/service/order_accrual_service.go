package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/ratelimit"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderRepository interface {
	SelectTenOrders(ctx context.Context) ([]model.Order, error)
}

type OrderAccrualRepository interface {
	UpdateOrderBalance(ctx context.Context, order model.Order) error
}

type OrderQueryAccrual interface {
	QueryUpdateOrder(orderNumber string) (*model.Order, error)
}

type OrderAccrualUseCase struct {
	orderRepository        OrderRepository
	orderAccrualRepository OrderAccrualRepository
	queryAccrual           OrderQueryAccrual
	logger                 *zap.Logger
}

func NewOrderAccrualService(repository OrderRepository, queryAccrual OrderQueryAccrual, orderAccrualRepository OrderAccrualRepository, logger *zap.Logger) *OrderAccrualUseCase {
	return &OrderAccrualUseCase{
		orderRepository:        repository,
		queryAccrual:           queryAccrual,
		orderAccrualRepository: orderAccrualRepository,
		logger:                 logger,
	}
}

func (oc *OrderAccrualUseCase) Run() {
	go func() {
		for {
			time.Sleep(300 * time.Millisecond)
			tenOrders, err := oc.orderRepository.SelectTenOrders(context.Background())
			if err != nil {
				continue
			}

			var wg sync.WaitGroup
			limiter := ratelimit.New(10)
			for _, order := range tenOrders {
				wg.Add(1)
				go oc.updateOrderBalance(&order, limiter, &wg)
			}

			wg.Wait()
		}
	}()
}

func (oc *OrderAccrualUseCase) updateOrderBalance(order *model.Order, rl ratelimit.Limiter, wg *sync.WaitGroup) {
	defer func() { wg.Done() }()

	rl.Take()
	updatedOrder, err := oc.queryAccrual.QueryUpdateOrder(order.Number)

	if errors.Is(err, apperrors.ErrRateLimit) {
		for i := 0; i < 6000; i++ {
			rl.Take()
		}
	}

	if err == nil {
		order.Accrual = updatedOrder.Accrual
		order.Status = updatedOrder.Status

		errUpdateOrderBalance := oc.orderAccrualRepository.UpdateOrderBalance(context.Background(), *order)
		if errUpdateOrderBalance != nil {
			oc.logger.Error("error while updating order balance", zap.Error(errUpdateOrderBalance))
		}
	}
}
