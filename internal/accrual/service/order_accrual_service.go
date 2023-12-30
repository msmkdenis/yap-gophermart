package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/avito-tech/go-transaction-manager/trm/manager"
	"github.com/avito-tech/go-transaction-manager/trm/settings"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
)

type OrderRepository interface {
	SelectTenOrders(ctx context.Context) ([]model.Order, error)
	UpdateOrder(ctx context.Context, order model.Order) error
}

type BalanceRepository interface {
	UpdateBalance(ctx context.Context, userLogin string, amount decimal.Decimal) error
}

type OrderQueryAccrual interface {
	QueryUpdateOrder(orderNumber string) (*model.Order, error)
}

type OrderAccrualUseCase struct {
	orderRepository   OrderRepository
	balanceRepository BalanceRepository
	queryAccrual      OrderQueryAccrual
	logger            *zap.Logger
	trManager         *manager.Manager
}

func NewOrderAccrualService(
	repository OrderRepository,
	balanceRepository BalanceRepository,
	queryAccrual OrderQueryAccrual,
	logger *zap.Logger,
	trManager *manager.Manager,
) *OrderAccrualUseCase {
	return &OrderAccrualUseCase{
		orderRepository:   repository,
		balanceRepository: balanceRepository,
		queryAccrual:      queryAccrual,
		logger:            logger,
		trManager:         trManager,
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

		s := pgxv5.MustSettings(
			settings.Must(settings.WithCancelable(true)),
			pgxv5.WithTxOptions(pgx.TxOptions{IsoLevel: pgx.RepeatableRead}),
		)

		errTransaction := oc.trManager.DoWithSettings(context.TODO(), s, func(ctx context.Context) error {
			errOrderUpdate := oc.orderRepository.UpdateOrder(ctx, *order)
			if errOrderUpdate != nil {
				oc.logger.Error("error while updating order", zap.Error(err))
				return errOrderUpdate
			}

			errBalanceUpdate := oc.balanceRepository.UpdateBalance(ctx, order.UserLogin, order.Accrual)
			if errBalanceUpdate != nil {
				oc.logger.Error("error while updating balance", zap.Error(err))
				return errBalanceUpdate
			}
			return nil
		})

		if errTransaction != nil {
			oc.logger.Error("error while updating order balance", zap.Error(err))
		}
	}
}
