package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/balance/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
)

// BalanceService mockgen --build_flags=--mod=mod -destination=internal/mocks/mock_balance_service.go -package=mock github.com/msmkdenis/yap-gophermart/internal/balance/handler BalanceService
type BalanceService interface {
	GetByUser(ctx context.Context, userLogin string) (*dto.BalanceResponse, error)
	Withdraw(ctx context.Context, orderNumber string, userLogin string, amount decimal.Decimal) error
	GetWithdrawals(ctx context.Context, userLogin string) ([]dto.WithdrawalResponse, error)
}

type BalanceHandler struct {
	balanceService BalanceService
	logger         *zap.Logger
	jwtAuth        *middleware.JWTAuth
}

func NewBalanceHandler(e *echo.Echo, service BalanceService, logger *zap.Logger, jwtAuth *middleware.JWTAuth) *BalanceHandler {
	handler := &BalanceHandler{
		balanceService: service,
		logger:         logger,
		jwtAuth:        jwtAuth,
	}

	protectedBalance := e.Group("/api/user", jwtAuth.JWTAuth())
	protectedBalance.GET("/balance", handler.GetBalance)
	protectedBalance.POST("/balance/withdraw", handler.Withdraw)
	protectedBalance.GET("/withdrawals", handler.GetWithdrawals)

	return handler
}

// @Summary       Get user balance
// @Description   Get the current balance of the user's loyalty points account.
// @Tags          Balance API
// @Produce       json
// @Success       200    {object}   dto.BalanceResponse
// @Failure       401
// @Failure       500
// @Security      JWT
// @Router        /api/user/balance [get]
func (h *BalanceHandler) GetBalance(c echo.Context) error {
	userLogin, ok := c.Get("userLogin").(string)
	if !ok {
		h.logger.Error("Internal server error", zap.Error(apperrors.ErrUnableToGetUserLoginFromContext))
		return c.NoContent(http.StatusInternalServerError)
	}

	balance, err := h.balanceService.GetByUser(c.Request().Context(), userLogin)
	if err != nil {
		h.logger.Error("Internal server error: unable to get balance", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, balance)
}

// @Summary       Get withdrawals list
// @Description   Get a list of withdrawals from a user's loyalty points account.
// @Tags          Balance API
// @Produce       json
// @Success       200    {array}     dto.WithdrawalResponse
// @Success       204
// @Failure       401
// @Failure       500
// @Security      JWT
// @Router        /api/user/withdrawals [get]
func (h *BalanceHandler) GetWithdrawals(c echo.Context) error {
	userLogin, ok := c.Get("userLogin").(string)
	if !ok {
		h.logger.Error("Internal server error", zap.Error(apperrors.ErrUnableToGetUserLoginFromContext))
		return c.NoContent(http.StatusInternalServerError)
	}

	withdrawals, err := h.balanceService.GetWithdrawals(c.Request().Context(), userLogin)
	if errors.Is(err, apperrors.ErrNoWithdrawals) {
		h.logger.Info("No withdrawals found", zap.Error(err))
		return c.NoContent(http.StatusNoContent)
	}

	if err != nil {
		h.logger.Error("Internal server error: unable to get withdrawals", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, withdrawals)
}

// @Summary       Withdrawal request
// @Description   Withdraw points from the loyalty points account to pay for a new order.
// @Tags          Balance API
// @Accept        json
// @Param         withdrawal   body       dto.BalanceWithdrawRequest   true   "Order number and withdrawal sum."
// @Success       200
// @Failure       401
// @Failure       402
// @Failure       422
// @Failure       500
// @Security      JWT
// @Router        /api/user/balance/withdraw [post]
func (h *BalanceHandler) Withdraw(c echo.Context) error {
	userLogin, ok := c.Get("userLogin").(string)
	if !ok {
		h.logger.Error("Internal server error", zap.Error(apperrors.ErrUnableToGetUserLoginFromContext))
		return c.NoContent(http.StatusInternalServerError)
	}

	header := c.Request().Header.Get("Content-Type")
	if header != "application/json" {
		msg := "Content-Type header is not application/json"
		h.logger.Error("StatusUnsupportedMediaType: " + msg)
		return c.String(http.StatusUnsupportedMediaType, msg)
	}

	request := new(dto.BalanceWithdrawRequest)
	if bindErr := c.Bind(request); bindErr != nil {
		h.logger.Warn("Unable to bind data", zap.Error(bindErr))
		return c.String(http.StatusBadRequest, "Bad request")
	}

	requestValidator := validator.New()
	errRegisterValidator := requestValidator.RegisterValidation("positive_withdraw", dto.PositiveWithdraw)
	if errRegisterValidator != nil {
		h.logger.Warn("Unable to register validator", zap.Error(errRegisterValidator))
	}

	if validateErr := requestValidator.Struct(request); validateErr != nil {
		h.logger.Warn("Bad Request: invalid request", zap.Error(validateErr))
		return c.String(http.StatusBadRequest, "Invalid request data")
	}

	err := h.balanceService.Withdraw(c.Request().Context(), request.OrderNumber, userLogin, request.Amount)

	if errors.Is(err, apperrors.ErrBadNumber) {
		h.logger.Error("Bad number", zap.Error(err))
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	if errors.Is(err, apperrors.ErrInsufficientFunds) {
		h.logger.Warn("Bad Request: insufficient funds", zap.Error(err))
		return c.NoContent(http.StatusPaymentRequired)
	}

	if err != nil {
		h.logger.Error("Internal server error", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}
