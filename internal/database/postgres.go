package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

type PostgresPool struct {
	DB     *pgxpool.Pool
	Logger *zap.Logger
}

func NewPostgresPool(connection string, logger *zap.Logger) (*PostgresPool, error) {
	dbPool, err := pgxpool.New(context.Background(), connection)
	if err != nil {
		return nil, apperrors.NewValueError(fmt.Sprintf("Unable to connect to database with connection %s", connection), utils.Caller(), err)
	}

	logger.Info(fmt.Sprintf("Connected to database with connection %s", connection))

	err = dbPool.Ping(context.Background())
	if err != nil {
		return nil, apperrors.NewValueError("Unable to ping database", utils.Caller(), err)
	}
	logger.Info(fmt.Sprintf("Pinged to database %s", dbPool.Config().ConnConfig.Database))

	return &PostgresPool{
		DB:     dbPool,
		Logger: logger,
	}, nil
}
