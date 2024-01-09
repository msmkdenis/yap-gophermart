package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/user/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/user/model"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

type UserRepository interface {
	Insert(ctx context.Context, u model.User) error
	SelectByLogin(ctx context.Context, login string) (*model.User, error)
}

type UserUseCase struct {
	repository UserRepository
	logger     *zap.Logger
}

func NewUserService(repository UserRepository, logger *zap.Logger) *UserUseCase {
	return &UserUseCase{
		repository: repository,
		logger:     logger,
	}
}

func (u *UserUseCase) Register(ctx context.Context, request dto.UserRegisterRequest) error {
	passHash, errHash := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if errHash != nil {
		return apperrors.NewValueError("unable to hash password", utils.Caller(), errHash)
	}

	userToSave := model.User{
		ID:       uuid.New().String(),
		Login:    request.Login,
		Password: passHash,
	}

	if err := u.repository.Insert(ctx, userToSave); err != nil {
		return fmt.Errorf("%s %w", utils.Caller(), err)
	}

	return nil
}

func (u *UserUseCase) Login(ctx context.Context, request dto.UserLoginRequest) error {
	user, err := u.repository.SelectByLogin(ctx, request.Login)
	if err != nil {
		return fmt.Errorf("%s %w", utils.Caller(), err)
	}

	if errPass := bcrypt.CompareHashAndPassword(user.Password, []byte(request.Password)); errPass != nil {
		return apperrors.ErrInvalidPassword
	}

	return nil
}
