package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/user/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

// UserService mockgen --build_flags=--mod=mod -destination=internal/mocks/mock_user_service.go -package=mock github.com/msmkdenis/yap-gophermart/internal/user/handler UserService
type UserService interface {
	Register(ctx context.Context, request dto.UserRegisterRequest) error
	Login(ctx context.Context, request dto.UserLoginRequest) error
}

type UserHandler struct {
	userService UserService
	jwtManager  *utils.JWTManager
	secret      string
	logger      *zap.Logger
}

func NewUserHandler(e *echo.Echo, service UserService, jwtManager *utils.JWTManager, secret string, logger *zap.Logger) *UserHandler {
	handler := &UserHandler{
		userService: service,
		jwtManager:  jwtManager,
		secret:      secret,
		logger:      logger,
	}

	e.POST("/api/user/register", handler.RegisterUser)
	e.POST("/api/user/login", handler.LoginUser)

	return handler
}

// @Summary       User registration
// @Description   User registration by login and password.
// @Tags          User API
// @Accept        json
// @Param         user   body       dto.UserRegisterRequest   true   "User login and password."
// @Success       200
// @Failure       400
// @Failure       409
// @Failure       500
// @Router        /api/user/register [post]
func (h *UserHandler) RegisterUser(c echo.Context) error {
	header := c.Request().Header.Get("Content-Type")
	if header != "application/json" {
		msg := "Content-Type header is not application/json"
		h.logger.Error("StatusUnsupportedMediaType: " + msg)
		return c.String(http.StatusUnsupportedMediaType, msg)
	}

	request := new(dto.UserRegisterRequest)
	if bindErr := c.Bind(request); bindErr != nil {
		h.logger.Warn("Unable to bind data", zap.Error(bindErr))
		return c.String(http.StatusBadRequest, "Bad request")
	}

	requestValidator := validator.New()
	if validateErr := requestValidator.Struct(request); validateErr != nil {
		h.logger.Warn("Bad Request: invalid request", zap.Error(validateErr))
		return c.String(http.StatusBadRequest, "Invalid request data")
	}

	err := h.userService.Register(c.Request().Context(), *request)

	if errors.Is(err, apperrors.ErrLoginAlreadyExists) {
		h.logger.Error("Non unique login", zap.Error(err))
		return c.NoContent(http.StatusConflict)
	}

	if err != nil {
		h.logger.Error("Unable to register user", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	errJWT := h.setAuthorizationHeader(c, request.Login)
	if errJWT != nil {
		h.logger.Error("Unable to set authorization header", zap.Error(errJWT))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

// @Summary       User authorization
// @Description   User authorization by login and password.
// @Tags          User API
// @Accept        json
// @Param         user   body       dto.UserLoginRequest   true   "User login and password."
// @Success       200
// @Failure       400
// @Failure       401
// @Failure       500
// @Router        /api/user/login [post]
func (h *UserHandler) LoginUser(c echo.Context) error {
	header := c.Request().Header.Get("Content-Type")
	if header != "application/json" {
		msg := "Content-Type header is not application/json"
		h.logger.Error("StatusUnsupportedMediaType: " + msg)
		return c.String(http.StatusUnsupportedMediaType, msg)
	}

	request := new(dto.UserLoginRequest)
	if bindErr := c.Bind(request); bindErr != nil {
		h.logger.Warn("Unable to bind data", zap.Error(bindErr))
		return c.String(http.StatusBadRequest, "Bad request")
	}

	requestValidator := validator.New()
	if validateErr := requestValidator.Struct(request); validateErr != nil {
		h.logger.Warn("Bad Request: invalid request", zap.Error(validateErr))
		return c.String(http.StatusBadRequest, "Invalid request data")
	}

	err := h.userService.Login(c.Request().Context(), *request)

	if errors.Is(err, apperrors.ErrInvalidPassword) {
		h.logger.Error("Invalid password", zap.Error(err))
		return c.NoContent(http.StatusUnauthorized)
	}

	if errors.Is(err, apperrors.ErrUserNotFound) {
		h.logger.Error("User not found", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	if err != nil {
		h.logger.Error("Unable to login user", zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	errCookie := h.setAuthorizationHeader(c, request.Login)
	if errCookie != nil {
		h.logger.Error("Unable to set authorization cookie", zap.Error(errCookie))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) setAuthorizationHeader(c echo.Context, login string) error {
	token, err := h.jwtManager.BuildJWTString(login)
	if err != nil {
		h.logger.Error("Unable to create token", zap.Error(err))
		return err
	}

	cookie := &http.Cookie{
		Name:  h.jwtManager.TokenName,
		Value: token,
	}
	c.SetCookie(cookie)

	return nil
}
