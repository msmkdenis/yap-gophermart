package service

import (
	"context"
	"fmt"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/balance/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/balance/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

type BalanceRepository interface {
	SelectByUserLogin(ctx context.Context, userLogin string) (*model.Balance, error)
	Withdraw(ctx context.Context, orderNumber string, userLogin string, amount decimal.Decimal) error
	SelectWithdrawalsByUserLogin(ctx context.Context, userLogin string) ([]model.Withdrawal, error)
}

type BalanceUseCase struct {
	repository BalanceRepository
	logger     *zap.Logger
}

func NewBalanceService(repository BalanceRepository, logger *zap.Logger) *BalanceUseCase {
	return &BalanceUseCase{
		repository: repository,
		logger:     logger,
	}
}

func (b *BalanceUseCase) GetByUser(ctx context.Context, userLogin string) (*dto.BalanceResponse, error) {
	balance, err := b.repository.SelectByUserLogin(ctx, userLogin)
	if err != nil {
		return nil, fmt.Errorf("%s %w", utils.Caller(), err)
	}

	balancerResponse := dto.MapToBalanceResponse(*balance)

	return &balancerResponse, nil
}

func (b *BalanceUseCase) Withdraw(ctx context.Context, orderNumber string, userLogin string, amount decimal.Decimal) error {
	errGoLuhn := goluhn.Validate(orderNumber)
	if errGoLuhn != nil {
		return apperrors.ErrBadNumber
	}

	err := b.repository.Withdraw(ctx, orderNumber, userLogin, amount)
	if err != nil {
		return fmt.Errorf("%s %w", utils.Caller(), err)
	}

	return nil
}

func (b *BalanceUseCase) GetWithdrawals(ctx context.Context, userLogin string) ([]dto.WithdrawalResponse, error) {
	withdrawals, err := b.repository.SelectWithdrawalsByUserLogin(ctx, userLogin)
	if err != nil {
		return nil, fmt.Errorf("%s %w", utils.Caller(), err)
	}

	withdrawalResponses := make([]dto.WithdrawalResponse, 0, len(withdrawals))
	for _, v := range withdrawals {
		withdrawalResponses = append(withdrawalResponses, dto.MapToWithdrawalResponse(v))
	}

	return withdrawalResponses, nil
}
