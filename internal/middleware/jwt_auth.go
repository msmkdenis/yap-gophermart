package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

type JWTAuth struct {
	jwtManager *utils.JWTManager
	logger     *zap.Logger
}

func InitJWTAuth(jwtManager *utils.JWTManager, logger *zap.Logger) *JWTAuth {
	j := &JWTAuth{
		jwtManager: jwtManager,
		logger:     logger,
	}
	return j
}

func (j *JWTAuth) JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Request().Cookie(j.jwtManager.TokenName)
			if err != nil {
				j.logger.Info("authentification failed", zap.Error(err))
				return c.NoContent(http.StatusUnauthorized)
			}
			userID, err := j.jwtManager.GetUserID(cookie.Value)
			if err != nil {
				j.logger.Info("authentification failed", zap.Error(err))
				return c.NoContent(http.StatusUnauthorized)
			}
			c.Set("userID", userID)
			j.logger.Info("authenticated", zap.String("userID", userID))
			return next(c)
		}
	}
}
