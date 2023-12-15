package user_repository

import (
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/user/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

//go:embed queries/insert_user.sql
var insertUser string

//go:embed queries/select_user_by_login.sql
var selectUserByLogin string

type PostgresUserRepository struct {
	PostgresPool *db.PostgresPool
	logger       *zap.Logger
}

func NewPostgresUserRepository(postgresPool *db.PostgresPool, logger *zap.Logger) *PostgresUserRepository {
	return &PostgresUserRepository{
		PostgresPool: postgresPool,
		logger:       logger,
	}
}

func (r *PostgresUserRepository) Insert(ctx context.Context, user model.User) error {

	_, err := r.PostgresPool.DB.Exec(ctx, insertUser, user.ID, user.Login, user.Password)

	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		return apperrors.ErrLoginAlreadyExists

	}

	return err
}

func (r *PostgresUserRepository) SelectByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	err := r.PostgresPool.DB.QueryRow(ctx, selectUserByLogin, login).Scan(&user.ID, &user.Login, &user.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apperrors.NewValueError("user not found", utils.Caller(), apperrors.ErrUserNotFound)
		} else {
			err = apperrors.NewValueError("query failed", utils.Caller(), err)
		}
		return nil, err
	}

	return &user, nil
}
