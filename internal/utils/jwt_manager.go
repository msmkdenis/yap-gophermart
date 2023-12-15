package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
)

type JWTManager struct {
	logger    *zap.Logger
	TokenName string
	secretKey string
}

const (
	tokenExp = time.Hour * 24
)

type claims struct {
	jwt.RegisteredClaims
	UserLogin string
}

func InitJWTManager(tokenName string, secretKey string, logger *zap.Logger) *JWTManager {
	j := &JWTManager{
		logger:    logger,
		TokenName: tokenName,
		secretKey: secretKey,
	}
	return j
}

func (j *JWTManager) BuildJWTString(userLogin string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		UserLogin: userLogin,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWTManager) GetUserID(tokenString string) (string, error) {
	claims := &claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperrors.NewValueError(fmt.Sprintf("unexpected signing method: %v", t.Header["alg"]), Caller(), errors.New("unexpected signing method"))
			}
			return []byte(j.secretKey), nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		j.logger.Warn("token is not valid", zap.Error(err))
		return "", apperrors.NewValueError("token is not valid", Caller(), errors.New("token is not valid"))
	}

	return claims.UserLogin, nil
}
