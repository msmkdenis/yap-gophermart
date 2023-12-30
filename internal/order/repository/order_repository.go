package repository

import (
	"context"
	_ "embed"
	"errors"

	trmpgx "github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // required for handling postgres errors
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/order/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

//go:embed queries/insert_order.sql
var insertOrder string

//go:embed queries/select_all_orders.sql
var selectAllOrders string

//go:embed queries/is_order_uploaded_by_user.sql
var isOrderUploadedByUser string

//go:embed queries/select_ten_orders.sql
var selectTenOrders string

//go:embed queries/update_order_by_order_number.sql
var updateOrderByNumber string

//go:embed queries/block_order_by_user.sql
var blockOrderByUser string

type PostgresOrderRepository struct {
	postgresPool *db.PostgresPool
	logger       *zap.Logger
	getter       *trmpgx.CtxGetter
}

func NewPostgresOrderRepository(postgresPool *db.PostgresPool, logger *zap.Logger) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		postgresPool: postgresPool,
		logger:       logger,
		getter:       trmpgx.DefaultCtxGetter,
	}
}

func (r *PostgresOrderRepository) UpdateOrder(ctx context.Context, order model.Order) error {
	conn := r.getter.DefaultTrOrDB(ctx, r.postgresPool.DB)

	batch := &pgx.Batch{}
	batch.Queue(blockOrderByUser, order.UserLogin)
	batch.Queue(updateOrderByNumber, order.Accrual, order.Status, order.Number)
	result := conn.SendBatch(ctx, batch)

	err := result.Close()
	if err != nil {
		return apperrors.NewValueError("close failed", utils.Caller(), err)
	}

	return err
}

func (r *PostgresOrderRepository) Insert(ctx context.Context, order model.Order) error {
	var isExists bool
	errExists := r.postgresPool.DB.QueryRow(ctx, isOrderUploadedByUser, order.Number, order.UserLogin).Scan(&isExists)
	if errExists != nil {
		return apperrors.NewValueError("query failed", utils.Caller(), errExists)
	}

	if isExists {
		return apperrors.ErrOrderUploadedByUser
	}

	_, err := r.postgresPool.DB.Exec(ctx, insertOrder, order.ID, order.Number, order.UserLogin, order.Status)

	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		return apperrors.ErrOrderUploadedByAnotherUser
	}

	return err
}

func (r *PostgresOrderRepository) SelectAll(ctx context.Context, userLogin string) ([]model.Order, error) {
	queryRows, err := r.postgresPool.DB.Query(ctx, selectAllOrders, userLogin)
	if err != nil {
		return nil, apperrors.NewValueError("query failed", utils.Caller(), err)
	}
	defer queryRows.Close()

	orders, err := pgx.CollectRows(queryRows, pgx.RowToStructByPos[model.Order])
	if err != nil {
		return nil, apperrors.NewValueError("unable to collect rows", utils.Caller(), err)
	}

	return orders, nil
}

func (r *PostgresOrderRepository) SelectTenOrders(ctx context.Context) ([]model.Order, error) {
	queryRows, err := r.postgresPool.DB.Query(ctx, selectTenOrders)
	if err != nil {
		return nil, apperrors.NewValueError("query failed", utils.Caller(), err)
	}
	defer queryRows.Close()

	orders, err := pgx.CollectRows(queryRows, pgx.RowToStructByPos[model.Order])
	if err != nil {
		return nil, apperrors.NewValueError("unable to collect rows", utils.Caller(), err)
	}

	if len(orders) == 0 {
		return nil, apperrors.NewValueError("no orders to process", utils.Caller(), apperrors.ErrNoOrders)
	}

	return orders, nil
}
