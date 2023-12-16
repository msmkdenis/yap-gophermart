package orderhandler

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
	"github.com/msmkdenis/yap-gophermart/internal/order/orderhandler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

// OrderService mockgen --build_flags=--mod=mod -destination=internal/mocks/mock_order_service.go -package=mock github.com/msmkdenis/yap-gophermart/internal/order/orderhandler OrderService
type OrderService interface {
	Upload(ctx context.Context, orderNumber string, userLogin string) error
	GetByUser(ctx context.Context, userLogin string) ([]dto.OrderResponse, error)
}

type OrderHandler struct {
	orderService OrderService
	logger       *zap.Logger
	jwtAuth      *middleware.JWTAuth
}

func NewOrderHandler(e *echo.Echo, service OrderService, logger *zap.Logger, jwtAuth *middleware.JWTAuth) *OrderHandler {
	handler := &OrderHandler{
		orderService: service,
		logger:       logger,
		jwtAuth:      jwtAuth,
	}

	protectedOrders := e.Group("/api/user/orders", jwtAuth.JWTAuth())
	protectedOrders.POST("", handler.AddOrder)
	protectedOrders.GET("", handler.GetOrders)

	return handler
}

func (h *OrderHandler) AddOrder(c echo.Context) error {
	userLogin, ok := c.Get("userLogin").(string)
	if !ok {
		h.logger.Error("Internal server error", zap.Error(apperrors.ErrUnableToGetUserLoginFromContext))
		return c.NoContent(http.StatusInternalServerError)
	}

	body, readErr := io.ReadAll(c.Request().Body)
	if readErr != nil {
		h.logger.Error("StatusBadRequest: unknown error", zap.Error(readErr))
		return c.String(http.StatusBadRequest, "Error: Unknown error, unable to read request")
	}

	if err := h.checkRequest(string(body)); err != nil {
		h.logger.Error("StatusBadRequest: unable to handle empty request", zap.Error(err))
		return c.String(http.StatusBadRequest, "Error: Unable to handle empty request")
	}

	err := h.orderService.Upload(c.Request().Context(), string(body), userLogin)

	if errors.Is(err, apperrors.ErrBadNumber) {
		h.logger.Error("Bad number", zap.Error(err))
		return c.NoContent(http.StatusUnprocessableEntity)
	}

	if errors.Is(err, apperrors.ErrOrderUploadedByUser) {
		h.logger.Error("Order already uploaded by user", zap.Error(err))
		return c.NoContent(http.StatusOK)
	}

	if errors.Is(err, apperrors.ErrOrderUploadedByAnotherUser) {
		h.logger.Error("Order already uploaded by another user", zap.Error(err))
		return c.NoContent(http.StatusConflict)
	}

	if err != nil {
		h.logger.Error("Unable to upload order", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusAccepted)
}

func (h *OrderHandler) GetOrders(c echo.Context) error {
	userLogin, ok := c.Get("userLogin").(string)
	if !ok {
		h.logger.Error("Internal server error", zap.Error(apperrors.ErrUnableToGetUserLoginFromContext))
		return c.NoContent(http.StatusInternalServerError)
	}

	orders, err := h.orderService.GetByUser(c.Request().Context(), userLogin)

	if errors.Is(err, apperrors.ErrNoOrders) {
		h.logger.Error("No orders", zap.Error(err))
		return c.NoContent(http.StatusNoContent)
	}

	if err != nil {
		h.logger.Error("Unknown error", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) checkRequest(s string) error {
	if len(s) == 0 {
		return apperrors.NewValueError("Unable to handle empty request", utils.Caller(), apperrors.ErrEmptyOrderRequest)
	}

	return nil
}
