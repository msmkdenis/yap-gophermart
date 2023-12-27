package repository

import (
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/balance/model"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

//go:embed queries/select_balance_by_user.sql
var selectBalanceByUser string

//go:embed queries/withdraw_from_balance_by_user.sql
var withdrawFromBalanceByUser string

//go:embed queries/insert_withdrawal.sql
var insertWithdrawal string

//go:embed queries/select_withdrawals_by_user.sql
var selectWithdrawalsByUser string

//go:embed queries/block_balance_by_user.sql
var blockBalanceByUser string

type PostgresBalanceRepository struct {
	postgresPool *db.PostgresPool
	logger       *zap.Logger
}

func NewPostgresBalanceRepository(postgresPool *db.PostgresPool, logger *zap.Logger) *PostgresBalanceRepository {
	return &PostgresBalanceRepository{
		postgresPool: postgresPool,
		logger:       logger,
	}
}

func (r *PostgresBalanceRepository) SelectByUserLogin(ctx context.Context, userLogin string) (*model.Balance, error) {
	var balance model.Balance
	err := r.postgresPool.DB.QueryRow(ctx, selectBalanceByUser, userLogin).
		Scan(&balance.ID, &balance.UserLogin, &balance.Current, &balance.Withdrawn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apperrors.ErrBalanceNotFound
		} else {
			err = apperrors.NewValueError("query failed", utils.Caller(), err)
		}
		return nil, err
	}

	return &balance, nil
}

func (r *PostgresBalanceRepository) SelectWithdrawalsByUserLogin(ctx context.Context, userLogin string) ([]model.Withdrawal, error) {
	queryRows, err := r.postgresPool.DB.Query(ctx, selectWithdrawalsByUser, userLogin)
	if err != nil {
		return nil, apperrors.NewValueError("query failed", utils.Caller(), err)
	}
	defer queryRows.Close()

	withdrawals, err := pgx.CollectRows(queryRows, pgx.RowToStructByPos[model.Withdrawal])
	if err != nil {
		return nil, apperrors.NewValueError("unable to collect rows", utils.Caller(), err)
	}

	if len(withdrawals) == 0 {
		return nil, apperrors.ErrNoWithdrawals
	}

	return withdrawals, nil
}

func (r *PostgresBalanceRepository) Withdraw(ctx context.Context, orderNumber string, userLogin string, amount decimal.Decimal) error {
	tx, err := r.postgresPool.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return apperrors.NewValueError("unable to start transaction", utils.Caller(), err)
	}
	defer tx.Rollback(ctx)

	block, err := tx.Prepare(ctx, "block", blockBalanceByUser)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	withdraw, err := tx.Prepare(ctx, "withdraw", withdrawFromBalanceByUser)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	saveWithdrawal, err := tx.Prepare(ctx, "saveWithdrawal", insertWithdrawal)
	if err != nil {
		return apperrors.NewValueError("unable to prepare query", utils.Caller(), err)
	}

	batch := &pgx.Batch{}
	batch.Queue(block.Name, userLogin)
	batch.Queue(withdraw.Name, amount, userLogin)
	batch.Queue(saveWithdrawal.Name, orderNumber, userLogin, amount)
	result := tx.SendBatch(ctx, batch)

	err = result.Close()
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.CheckViolation {
		return apperrors.ErrInsufficientFunds
	}

	if err != nil {
		return apperrors.NewValueError("close failed", utils.Caller(), err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return apperrors.NewValueError("commit failed", utils.Caller(), err)
	}

	return nil
}
