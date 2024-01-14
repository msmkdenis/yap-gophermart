package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/balance/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/config"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
	mock "github.com/msmkdenis/yap-gophermart/internal/mocks"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

var cfgMock = &config.Config{
	Address:              "localhost:8000",
	DatabaseURI:          "user=postgres password=postgres host=localhost database=yap-gophermart sslmode=disable",
	AccrualSystemAddress: "http://localhost:8080",
	Secret:               "supersecretkey",
	TokenName:            "token",
}

type BalanceHandlersSuite struct {
	suite.Suite
	h              *BalanceHandler
	balanceService *mock.MockBalanceService
	echo           *echo.Echo
	ctrl           *gomock.Controller
	jwtManager     *utils.JWTManager
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(BalanceHandlersSuite))
}

func (b *BalanceHandlersSuite) SetupTest() {
	logger, _ := zap.NewProduction()
	jwtManager := utils.InitJWTManager(cfgMock.TokenName, cfgMock.Secret, logger)
	jwtAuth := middleware.InitJWTAuth(jwtManager, logger)
	b.jwtManager = jwtManager
	b.ctrl = gomock.NewController(b.T())
	b.echo = echo.New()
	b.balanceService = mock.NewMockBalanceService(b.ctrl)
	b.h = NewBalanceHandler(b.echo, b.balanceService, logger, jwtAuth)
}

func (b *BalanceHandlersSuite) TestGetBalance() {
	login := "awesome_login"

	cookie, errCookie := b.createCookie(login)
	require.NoError(b.T(), errCookie)

	balanceResponse := &dto.BalanceResponse{
		Current:   decimal.NewFromInt(100),
		Withdrawn: decimal.NewFromInt(0),
	}

	response, errMarshal := json.Marshal(balanceResponse)
	require.NoError(b.T(), errMarshal)

	testCases := []struct {
		name         string
		method       string
		header       http.Header
		cookie       *http.Cookie
		path         string
		prepare      func()
		expectedCode int
		expectedBody []byte
	}{
		{
			name:   "Unauthorized - 401",
			method: http.MethodGet,
			path:   "http://localhost:8000/api/user/balance",
			prepare: func() {
				b.balanceService.EXPECT().GetByUser(gomock.Any(), login).Times(0)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "Success - 200",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/balance",
			prepare: func() {
				b.balanceService.EXPECT().GetByUser(gomock.Any(), login).Times(1).Return(balanceResponse, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: response,
		},
		{
			name:   "InternalServerError - 500",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/balance",
			prepare: func() {
				b.balanceService.EXPECT().GetByUser(gomock.Any(), login).Times(1).Return(nil, errors.New("some error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range testCases {
		b.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}

			w := httptest.NewRecorder()
			b.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
			if test.expectedBody != nil {
				var result dto.BalanceResponse
				jsonErr := json.Unmarshal(w.Body.Bytes(), &result)
				require.NoError(t, jsonErr)

				var expected dto.BalanceResponse
				jsonErrExp := json.Unmarshal(test.expectedBody, &expected)
				require.NoError(t, jsonErrExp)

				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, "", w.Body.String())
			}
		})
	}
}

func (b *BalanceHandlersSuite) TestGetWithdrawals() {
	login := "awesome_login"

	cookie, errCookie := b.createCookie(login)
	require.NoError(b.T(), errCookie)

	withdrawalResponse := []dto.WithdrawalResponse{
		{
			OrderNumber: "123",
			Amount:      decimal.NewFromInt(100),
			ProcessedAt: time.Now().Format(time.RFC3339),
		},
		{
			OrderNumber: "456",
			Amount:      decimal.NewFromInt(200),
			ProcessedAt: time.Now().Format(time.RFC3339),
		},
	}

	response, errMarshal := json.Marshal(withdrawalResponse)
	require.NoError(b.T(), errMarshal)

	testCases := []struct {
		name         string
		method       string
		header       http.Header
		cookie       *http.Cookie
		path         string
		prepare      func()
		expectedCode int
		expectedBody []byte
	}{
		{
			name:   "Unauthorized - 401",
			method: http.MethodGet,
			path:   "http://localhost:8000/api/user/withdrawals",
			prepare: func() {
				b.balanceService.EXPECT().GetWithdrawals(gomock.Any(), login).Times(0)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "Success - 200",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/withdrawals",
			prepare: func() {
				b.balanceService.EXPECT().GetWithdrawals(gomock.Any(), login).Times(1).Return(withdrawalResponse, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: response,
		},
		{
			name:   "NoContent - 204",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/withdrawals",
			prepare: func() {
				b.balanceService.EXPECT().GetWithdrawals(gomock.Any(), login).Times(1).Return(nil, apperrors.ErrNoWithdrawals)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "InternalServerError - 500",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/withdrawals",
			prepare: func() {
				b.balanceService.EXPECT().GetWithdrawals(gomock.Any(), login).Times(1).Return(nil, errors.New("some error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range testCases {
		b.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}

			w := httptest.NewRecorder()
			b.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
			if test.expectedBody != nil {
				var result []dto.WithdrawalResponse
				jsonErr := json.Unmarshal(w.Body.Bytes(), &result)
				require.NoError(t, jsonErr)

				var expected []dto.WithdrawalResponse
				jsonErrExp := json.Unmarshal(test.expectedBody, &expected)
				require.NoError(t, jsonErrExp)

				assert.Equal(t, expected, result)
			} else {
				assert.Equal(t, "", w.Body.String())
			}
		})
	}
}

func (b *BalanceHandlersSuite) TestWithdraw() {
	login := "awesome_login"

	cookie, errCookie := b.createCookie(login)
	require.NoError(b.T(), errCookie)

	validWithdrawRequest := dto.BalanceWithdrawRequest{
		OrderNumber: "123",
		Amount:      decimal.NewFromInt(100),
	}

	validReq, errMarshal := json.Marshal(validWithdrawRequest)
	require.NoError(b.T(), errMarshal)

	invalidWithdrawRequest := dto.BalanceWithdrawRequest{
		Amount: decimal.NewFromInt(100),
	}

	invalidReq, errMarshal := json.Marshal(invalidWithdrawRequest)
	require.NoError(b.T(), errMarshal)

	testCases := []struct {
		name         string
		method       string
		header       http.Header
		cookie       *http.Cookie
		path         string
		prepare      func()
		expectedCode int
		body         string
	}{
		{
			name:   "Unauthorized - 401",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "UnprocessableEntity - 422",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {"application/json"}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apperrors.ErrBadNumber)
			},
			expectedCode: http.StatusUnprocessableEntity,
			body:         string(validReq),
		},
		{
			name:   "UnsupportedMediaType - 415",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {""}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedCode: http.StatusUnsupportedMediaType,
			body:         string(validReq),
		},
		{
			name:   "Bad Request - 400",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {"application/json"}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedCode: http.StatusBadRequest,
			body:         string(invalidReq),
		},
		{
			name:   "PaymentRequired - 402",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {"application/json"}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(apperrors.ErrInsufficientFunds)
			},
			expectedCode: http.StatusPaymentRequired,
			body:         string(validReq),
		},
		{
			name:   "InternalServerError - 500",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {"application/json"}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(errors.New("some error"))
			},
			expectedCode: http.StatusInternalServerError,
			body:         string(validReq),
		},
		{
			name:   "Success - 200",
			method: http.MethodPost,
			path:   "http://localhost:8000/api/user/balance/withdraw",
			header: map[string][]string{"Content-Type": {"application/json"}},
			cookie: cookie,
			prepare: func() {
				b.balanceService.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			expectedCode: http.StatusOK,
			body:         string(validReq),
		},
	}

	for _, test := range testCases {
		b.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}
			request.Header.Set("Content-Type", test.header.Get("Content-Type"))
			w := httptest.NewRecorder()
			b.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
		})
	}
}

func (b *BalanceHandlersSuite) createCookie(login string) (*http.Cookie, error) {
	token, err := b.jwtManager.BuildJWTString(login)

	cookie := &http.Cookie{
		Name:  b.jwtManager.TokenName,
		Value: token,
	}

	return cookie, err
}
