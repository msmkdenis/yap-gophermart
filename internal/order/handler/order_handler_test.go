package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/config"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
	mock "github.com/msmkdenis/yap-gophermart/internal/mocks"
	"github.com/msmkdenis/yap-gophermart/internal/order/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

var cfgMock = &config.Config{
	Address:              "localhost:8000",
	DatabaseURI:          "user=postgres password=postgres host=localhost database=yap-gophermart sslmode=disable",
	AccrualSystemAddress: "http://localhost:8080",
	Secret:               "supersecretkey",
	TokenName:            "token",
}

type OrderHandlersSuite struct {
	suite.Suite
	h            *OrderHandler
	orderService *mock.MockOrderService
	echo         *echo.Echo
	ctrl         *gomock.Controller
	jwtManager   *utils.JWTManager
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(OrderHandlersSuite))
}

func (o *OrderHandlersSuite) SetupTest() {
	logger, _ := zap.NewProduction()
	jwtManager := utils.InitJWTManager(cfgMock.TokenName, cfgMock.Secret, logger)
	jwtAuth := middleware.InitJWTAuth(jwtManager, logger)
	o.jwtManager = jwtManager
	o.ctrl = gomock.NewController(o.T())
	o.echo = echo.New()
	o.orderService = mock.NewMockOrderService(o.ctrl)
	o.h = NewOrderHandler(o.echo, o.orderService, logger, jwtAuth)
}

func (o *OrderHandlersSuite) TestAddOrder() {
	login := "awesome_login"

	cookie, errCookie := o.createCookie(login)
	require.NoError(o.T(), errCookie)

	validNumber := "123"
	invalidNumber := "0123"

	testCases := []struct {
		name               string
		method             string
		header             http.Header
		cookie             *http.Cookie
		body               string
		path               string
		prepare            func()
		expectedCode       int
		expectedBody       string
		expectedLogin      string
		expectedCookieName string
	}{
		{
			name:   "Unauthorized - 401",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			body:   validNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Times(0)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "",
		},
		{
			name:   "Success - 202",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   validNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Return(nil)
			},
			expectedCode: http.StatusAccepted,
			expectedBody: "",
		},
		{
			name:   "Bad number - 422",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   invalidNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), invalidNumber, login).Return(apperrors.ErrBadNumber)
			},
			expectedCode: http.StatusUnprocessableEntity,
			expectedBody: "",
		},
		{
			name:   "OrderUploadedByUser - 200",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   validNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Return(apperrors.ErrOrderUploadedByUser)
			},
			expectedCode: http.StatusOK,
			expectedBody: "",
		},
		{
			name:   "OrderUploadedByAnotherUser - 409",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   validNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Return(apperrors.ErrOrderUploadedByAnotherUser)
			},
			expectedCode: http.StatusConflict,
			expectedBody: "",
		},
		{
			name:   "InternalServerError - 500",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   validNumber,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Return(errors.New("some error"))
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: "",
		},
		{
			name:   "BadRequest - 400",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"plain/text"}},
			cookie: cookie,
			body:   "",
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().Upload(gomock.Any(), validNumber, login).Times(0)
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Error: Unable to handle empty request",
		},
	}

	for _, test := range testCases {
		o.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}

			request.Header.Set("Content-Type", test.header.Get("Content-Type"))

			w := httptest.NewRecorder()
			o.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}

func (o *OrderHandlersSuite) TestGetOrders() {
	login := "awesome_login"

	cookie, errCookie := o.createCookie(login)
	require.NoError(o.T(), errCookie)

	ordersResponse := []dto.OrderResponse{
		{
			Number:     "123",
			Status:     "NEW",
			UploadedAt: "2020-01-01T00:00:00Z",
		},
		{
			Number:     "456",
			Status:     "NEW",
			UploadedAt: "2021-01-01T00:00:00Z",
		},
	}

	response, errMarshal := json.Marshal(ordersResponse)
	require.NoError(o.T(), errMarshal)

	testCases := []struct {
		name               string
		method             string
		header             http.Header
		cookie             *http.Cookie
		path               string
		prepare            func()
		expectedCode       int
		expectedBody       []byte
		expectedLogin      string
		expectedCookieName string
	}{
		{
			name:   "Unauthorized - 401",
			method: http.MethodGet,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().GetByUser(gomock.Any(), login).Times(0)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "Success - 200",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().GetByUser(gomock.Any(), login).Times(1).Return(ordersResponse, nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: response,
		},
		{
			name:   "NoContent - 204",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().GetByUser(gomock.Any(), login).Times(1).Return(nil, apperrors.ErrNoOrders)
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "InternalServerError - 500",
			method: http.MethodGet,
			cookie: cookie,
			path:   "http://localhost:8000/api/user/orders",
			prepare: func() {
				o.orderService.EXPECT().GetByUser(gomock.Any(), login).Times(1).Return(nil, errors.New("some error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range testCases {
		o.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}

			w := httptest.NewRecorder()
			o.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
			if test.expectedBody != nil {
				var result []dto.OrderResponse
				jsonErr := json.Unmarshal(w.Body.Bytes(), &result)
				require.NoError(t, jsonErr)

				var expectedBody []dto.OrderResponse
				jsonErrExp := json.Unmarshal(test.expectedBody, &expectedBody)
				require.NoError(t, jsonErrExp)

				assert.Equal(t, expectedBody, result)
			} else {
				assert.Equal(t, "", w.Body.String())
			}
		})
	}
}

func (o *OrderHandlersSuite) createCookie(login string) (*http.Cookie, error) {
	token, err := o.jwtManager.BuildJWTString(login)

	cookie := &http.Cookie{
		Name:  o.jwtManager.TokenName,
		Value: token,
	}

	return cookie, err
}
