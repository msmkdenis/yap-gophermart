package handler

import (
	"encoding/json"
	"io"
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
	mock "github.com/msmkdenis/yap-gophermart/internal/mocks"
	"github.com/msmkdenis/yap-gophermart/internal/user/handler/dto"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

var cfgMock = &config.Config{
	Address:              "localhost:8000",
	DatabaseURI:          "user=postgres password=postgres host=localhost database=yap-gophermart sslmode=disable",
	AccrualSystemAddress: "http://localhost:8080",
	Secret:               "supersecretkey",
	TokenName:            "token",
}

type UserHandlersSuite struct {
	suite.Suite
	h           *UserHandler
	userService *mock.MockUserService
	echo        *echo.Echo
	ctrl        *gomock.Controller
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(UserHandlersSuite))
}

func (s *UserHandlersSuite) SetupTest() {
	logger, _ := zap.NewProduction()
	jwtManager := utils.InitJWTManager(cfgMock.TokenName, cfgMock.Secret, logger)
	s.ctrl = gomock.NewController(s.T())
	s.echo = echo.New()
	s.userService = mock.NewMockUserService(s.ctrl)
	s.h = NewUserHandler(s.echo, s.userService, jwtManager, cfgMock.Secret, logger)
}

func (s *UserHandlersSuite) TestRegisterUser() {
	invalidRegisterRequest := dto.UserRegisterRequest{
		Login:    "awesome_login",
		Password: "",
	}

	validRegisterRequest := dto.UserRegisterRequest{
		Login:    "awesome_login",
		Password: "awesome_password",
	}

	invalidRegisterRequestTaskJSON, err := json.Marshal(invalidRegisterRequest)
	require.NoError(s.T(), err)

	validRegisterRequestTaskJSON, err := json.Marshal(validRegisterRequest)
	require.NoError(s.T(), err)

	testCases := []struct {
		name               string
		method             string
		header             http.Header
		body               string
		path               string
		prepare            func()
		expectedCode       int
		expectedBody       string
		expectedLogin      string
		expectedCookieName string
	}{
		{
			name:   "Success - 200 OK",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"application/json"}},
			body:   string(validRegisterRequestTaskJSON),
			path:   "http://localhost:8000/api/user/register",
			prepare: func() {
				s.userService.EXPECT().Register(gomock.Any(), validRegisterRequest).Times(1).Return(nil)
			},
			expectedCode:       http.StatusOK,
			expectedBody:       "",
			expectedLogin:      validRegisterRequest.Login,
			expectedCookieName: cfgMock.TokenName,
		},
		{
			name:         "BadRequest - invalid request",
			method:       http.MethodPost,
			header:       map[string][]string{"Content-Type": {"application/json"}},
			body:         string(invalidRegisterRequestTaskJSON),
			path:         "http://localhost:8000/api/user/register",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid request data",
		},
		{
			name:         "BadRequest - invalid request",
			method:       http.MethodPost,
			header:       map[string][]string{"Content-Type": {""}},
			body:         string(invalidRegisterRequestTaskJSON),
			path:         "http://localhost:8000/api/user/register",
			expectedCode: http.StatusUnsupportedMediaType,
			expectedBody: "Content-Type header is not application/json",
		},
		{
			name:   "Non unique login - 409 Status conflict",
			method: http.MethodPost,
			header: map[string][]string{"Content-Type": {"application/json"}},
			body:   string(validRegisterRequestTaskJSON),
			path:   "http://localhost:8000/api/user/register",
			prepare: func() {
				s.userService.EXPECT().Register(gomock.Any(), validRegisterRequest).Times(1).Return(apperrors.ErrLoginAlreadyExists)
			},
			expectedCode: http.StatusConflict,
			expectedBody: "",
		},
	}

	for _, test := range testCases {
		s.T().Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.header.Get("Content-Type"))
			w := httptest.NewRecorder()
			s.echo.ServeHTTP(w, request)

			assert.Equal(t, test.expectedCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())

			switch test.expectedCode {
			case http.StatusOK:
				cookie := w.Result().Cookies()[0]
				defer func(Body io.ReadCloser) {
					errCloser := Body.Close()
					require.NoError(t, errCloser)
				}(w.Result().Body)

				assert.NotEmpty(t, cookie)
				assert.Equal(t, test.expectedCookieName, cookie.Name)

				login, errCookieParse := s.h.jwtManager.GetUserID(cookie.Value)
				assert.NoError(t, errCookieParse)
				assert.Equal(t, test.expectedLogin, login)
			default:
				cookies := w.Result().Cookies()
				assert.Empty(t, cookies)
			}
		})
	}
}
