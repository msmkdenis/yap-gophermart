package repository

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

//go:embed queries/block_order_by_user.sql
var blockOrderByUser string

//go:embed queries/update_order_by_order_number.sql
var updateOrderByNumber string

//go:embed queries/block_balance_by_user.sql
var blockBalanceByUser string

//go:embed queries/bonus_accrual.sql
var bonusAccrual string

type PostgresOrderAccrualRepository struct {
	postgresPool *db.PostgresPool
	logger       *zap.Logger
}

func NewOrderAccrualRepository(postgresPool *db.PostgresPool, logger *zap.Logger) *PostgresOrderAccrualRepository {
	return &PostgresOrderAccrualRepository{
		postgresPool: postgresPool,
		logger:       logger,
	}
}

func (r *PostgresOrderAccrualRepository) UpdateOrderBalance(ctx context.Context, order model.Order, userLogin string, amount decimal.Decimal) error {
	_, err := r.postgresPool.DB.Exec(ctx, updateOrderByNumber, order.Accrual, order.Status, order.Number)

	tx, err := r.postgresPool.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return apperrors.NewValueError("unable to start transaction", utils.Caller(), err)
	}
	defer tx.Rollback(ctx)

	blockOrder, err := tx.Prepare(ctx, "blockOrder", blockOrderByUser)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	updateOrder, err := tx.Prepare(ctx, "updateOrder", updateOrderByNumber)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	blockBalance, err := tx.Prepare(ctx, "blockBalance", blockBalanceByUser)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	makeAccrual, err := tx.Prepare(ctx, "makeAccrual", bonusAccrual)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	batch := &pgx.Batch{}
	batch.Queue(blockOrder.Name, userLogin)
	batch.Queue(blockBalance.Name, userLogin)
	batch.Queue(updateOrder.Name, order.Accrual, order.Status, order.Number)
	batch.Queue(makeAccrual.Name, amount, userLogin)
	result := tx.SendBatch(ctx, batch)

	err = result.Close()
	if err != nil {
		return apperrors.NewValueError("close failed", utils.Caller(), err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return apperrors.NewValueError("commit failed", utils.Caller(), err)
	}

	return nil
}
