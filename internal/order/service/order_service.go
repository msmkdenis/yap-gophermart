package service

import (
	"context"
	"fmt"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/order/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

type OrderRepository interface {
	Insert(ctx context.Context, order model.Order) error
	SelectAll(ctx context.Context, userLogin string) ([]model.Order, error)
}

type OrderUseCase struct {
	repository OrderRepository
	logger     *zap.Logger
}

func NewOrderService(repository OrderRepository, logger *zap.Logger) *OrderUseCase {
	return &OrderUseCase{
		repository: repository,
		logger:     logger,
	}
}

func (u *OrderUseCase) Upload(ctx context.Context, orderNumber string, userLogin string) error {
	errGoLuhn := goluhn.Validate(orderNumber)
	if errGoLuhn != nil {
		return apperrors.ErrBadNumber
	}

	order := model.Order{
		ID:        uuid.New().String(),
		Number:    orderNumber,
		UserLogin: userLogin,
		Status:    "NEW",
	}

	if err := u.repository.Insert(ctx, order); err != nil {
		return fmt.Errorf("%s %w", utils.Caller(), err)
	}

	return nil
}

func (u *OrderUseCase) GetByUser(ctx context.Context, userLogin string) ([]dto.OrderResponse, error) {
	orders, err := u.repository.SelectAll(ctx, userLogin)
	if err != nil {
		return nil, fmt.Errorf("%s %w", utils.Caller(), err)
	}

	if len(orders) == 0 {
		return nil, apperrors.ErrNoOrders
	}

	orderResponse := make([]dto.OrderResponse, 0, len(orders))
	for _, v := range orders {
		orderResponse = append(orderResponse, dto.MapToOrderResponse(v))
	}

	return orderResponse, nil
}
