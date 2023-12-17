package orderservice

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderAccrualRepository interface {
	UpdateOrder(ctx context.Context, order model.Order) error
	SelectTenOrders(ctx context.Context) ([]model.Order, error)
}

type BalanceAccrualRepository interface {
	UpdateBalance(ctx context.Context, userLogin string, amount decimal.Decimal) error
}

type OrderQueryAccrual interface {
	QueryUpdateOrder(orderNumber string) (*model.Order, error)
}

type OrderAccrualUseCase struct {
	orderRepository   OrderAccrualRepository
	balanceRepository BalanceAccrualRepository
	queryAccrual      OrderQueryAccrual
	logger            *zap.Logger
}

func NewOrderAccrualService(repository OrderAccrualRepository, queryAccrual OrderQueryAccrual, balanceRepository BalanceAccrualRepository, logger *zap.Logger) *OrderAccrualUseCase {
	return &OrderAccrualUseCase{
		orderRepository:   repository,
		queryAccrual:      queryAccrual,
		balanceRepository: balanceRepository,
		logger:            logger,
	}
}

func (oc *OrderAccrualUseCase) Run() {
	go func() {
		for {
			time.Sleep(300 * time.Millisecond)
			tenOrders, err := oc.orderRepository.SelectTenOrders(context.Background())
			if err != nil {
				// oc.logger.Info("error while processing accrual", zap.Error(err))
				continue
			}

			var wg sync.WaitGroup
			limiter := ratelimit.New(10) // 10 timeslots per second
			for _, order := range tenOrders {
				wg.Add(1)
				go oc.updateOrder(&order, limiter, &wg)
			}

			wg.Wait()
		}
	}()
}

func (oc *OrderAccrualUseCase) updateOrder(o *model.Order, rl ratelimit.Limiter, wg *sync.WaitGroup) {
	defer func() { wg.Done() }()

	rl.Take()
	updatedOrder, err := oc.queryAccrual.QueryUpdateOrder(o.Number)

	if err == nil {
		o.Accrual = updatedOrder.Accrual
		o.Status = updatedOrder.Status
		errUpdateOrder := oc.orderRepository.UpdateOrder(context.Background(), *o)
		if errUpdateOrder != nil {
			oc.logger.Error("error while updating order", zap.Error(errUpdateOrder))
		}

		errUpdateBalance := oc.balanceRepository.UpdateBalance(context.Background(), o.UserLogin, o.Accrual)
		if errUpdateBalance != nil {
			oc.logger.Error("error while updating order", zap.Error(errUpdateOrder))
		}
	}

	if errors.Is(err, apperrors.ErrRateLimit) {
		for i := 0; i < 60000; i++ {
			rl.Take()
		}
	}
}
