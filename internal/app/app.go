package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msmkdenis/yap-gophermart/internal/balance/balancehandler"
	"github.com/msmkdenis/yap-gophermart/internal/balance/balancerepository"
	"github.com/msmkdenis/yap-gophermart/internal/balance/balanceservice"

	"github.com/msmkdenis/yap-gophermart/internal/order/accrual"
	"github.com/shopspring/decimal"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/config"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
	"github.com/msmkdenis/yap-gophermart/internal/order/orderhandler"
	"github.com/msmkdenis/yap-gophermart/internal/order/orderrepository"
	"github.com/msmkdenis/yap-gophermart/internal/order/orderservice"
	"github.com/msmkdenis/yap-gophermart/internal/user/userhandler"
	"github.com/msmkdenis/yap-gophermart/internal/user/userrepository"
	"github.com/msmkdenis/yap-gophermart/internal/user/userservice"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

func GophermartRun() {
	cfg := *config.NewConfig()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Unable to initialize zap logger", err)
	}

	decimal.MarshalJSONWithoutQuotes = true

	jwtManager := utils.InitJWTManager(cfg.TokenName, cfg.Secret, logger)
	postgresPool := initPostgresPool(&cfg, logger)

	userRepository := userrepository.NewPostgresUserRepository(postgresPool, logger)
	userService := userservice.NewUserService(userRepository, logger)

	orderRepository := orderrepository.NewPostgresOrderRepository(postgresPool, logger)
	orderService := orderservice.NewOrderService(orderRepository, logger)

	balanceRepository := balancerepository.NewPostgresBalanceRepository(postgresPool, logger)
	balanceService := balanceservice.NewBalanceService(balanceRepository, logger)

	orderAccrual := accrual.NewOrderAccrual(cfg.AccrualSystemAddress, logger)
	orderservice.NewOrderAccrualService(orderRepository, orderAccrual, balanceRepository, logger).Run()

	requestLogger := middleware.InitRequestLogger(logger)
	jwtAuth := middleware.InitJWTAuth(jwtManager, logger)

	e := echo.New()

	e.Use(requestLogger.RequestLogger())
	e.Use(middleware.Compress())
	e.Use(middleware.Decompress())

	userhandler.NewUserHandler(e, userService, jwtManager, cfg.Secret, logger)
	orderhandler.NewOrderHandler(e, orderService, logger, jwtAuth)
	balancehandler.NewBalanceHandler(e, balanceService, logger, jwtAuth)

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 5*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		if errShutdown := e.Shutdown(shutdownCtx); errShutdown != nil {
			e.Logger.Fatal(errShutdown)
		}
		serverStopCtx()
	}()

	errStart := e.Start(cfg.Address)
	if errStart != nil && !errors.Is(errStart, http.ErrServerClosed) {
		log.Fatal(err)
	}

	<-serverCtx.Done()
}

func initPostgresPool(cfg *config.Config, logger *zap.Logger) *db.PostgresPool {
	postgresPool, err := db.NewPostgresPool(cfg.DatabaseURI, logger)
	if err != nil {
		logger.Fatal("Unable to connect to database", zap.Error(err))
	}

	migrations, err := db.NewMigrations(cfg.DatabaseURI, logger)
	if err != nil {
		logger.Fatal("Unable to create migrations", zap.Error(err))
	}

	err = migrations.MigrateUp()
	if err != nil {
		logger.Fatal("Unable to up migrations", zap.Error(err))
	}

	logger.Info("Connected to database", zap.String("DSN", cfg.DatabaseURI))
	return postgresPool
}
