package gophermart

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	accrualHttp "github.com/msmkdenis/yap-gophermart/internal/accrual/http"
	accrualOrderRepository "github.com/msmkdenis/yap-gophermart/internal/accrual/repository"
	accrualService "github.com/msmkdenis/yap-gophermart/internal/accrual/service"
	balanceHandler "github.com/msmkdenis/yap-gophermart/internal/balance/handler"
	balanceRepository "github.com/msmkdenis/yap-gophermart/internal/balance/repository"
	balanceService "github.com/msmkdenis/yap-gophermart/internal/balance/service"
	"github.com/msmkdenis/yap-gophermart/internal/config"
	db "github.com/msmkdenis/yap-gophermart/internal/database"
	"github.com/msmkdenis/yap-gophermart/internal/middleware"
	orderHandler "github.com/msmkdenis/yap-gophermart/internal/order/handler"
	orderRepository "github.com/msmkdenis/yap-gophermart/internal/order/repository"
	orderService "github.com/msmkdenis/yap-gophermart/internal/order/service"
	userHandler "github.com/msmkdenis/yap-gophermart/internal/user/handler"
	userRepository "github.com/msmkdenis/yap-gophermart/internal/user/repository"
	userService "github.com/msmkdenis/yap-gophermart/internal/user/service"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

func Run() {
	cfg := *config.NewConfig()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Unable to initialize zap logger", err)
	}

	decimal.MarshalJSONWithoutQuotes = true

	jwtManager := utils.InitJWTManager(cfg.TokenName, cfg.Secret, logger)
	postgresPool := initPostgresPool(&cfg, logger)

	userRepo := userRepository.NewPostgresUserRepository(postgresPool, logger)
	userServ := userService.NewUserService(userRepo, logger)

	orderRepo := orderRepository.NewPostgresOrderRepository(postgresPool, logger)
	orderServ := orderService.NewOrderService(orderRepo, logger)

	balanceRepo := balanceRepository.NewPostgresBalanceRepository(postgresPool, logger)
	balanceServ := balanceService.NewBalanceService(balanceRepo, logger)

	orderAccrual := accrualHttp.NewOrderAccrual(cfg.AccrualSystemAddress, logger)
	orderAccrualRepo := accrualOrderRepository.NewOrderAccrualRepository(postgresPool, logger)
	accrualService.NewOrderAccrualService(orderRepo, orderAccrual, orderAccrualRepo, logger).Run()

	requestLogger := middleware.InitRequestLogger(logger)
	jwtAuth := middleware.InitJWTAuth(jwtManager, logger)

	e := echo.New()

	e.Use(requestLogger.RequestLogger())
	e.Use(middleware.Compress())
	e.Use(middleware.Decompress())

	userHandler.NewUserHandler(e, userServ, jwtManager, cfg.Secret, logger)
	orderHandler.NewOrderHandler(e, orderServ, logger, jwtAuth)
	balanceHandler.NewBalanceHandler(e, balanceServ, logger, jwtAuth)

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit

		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
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
