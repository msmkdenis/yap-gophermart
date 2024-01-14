package accrual

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/caarlos0/env/v10"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/middleware"
)

type Config struct {
	Address string `env:"ACCRUAL_RUN_ADDRESS" envDefault:"0.0.0.0:8000"`
}

func Run() {
	config := &Config{}
	flag.StringVar(&config.Address, "a", "0.0.0.0:8000", "Адрес и порт запуска сервиса")

	if err := env.Parse(config); err != nil {
		fmt.Printf("%+v\n", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Unable to initialize zap logger", err)
	}

	decimal.MarshalJSONWithoutQuotes = true
	e := echo.New()

	requestLogger := middleware.InitRequestLogger(logger)
	e.Use(requestLogger.RequestLogger())

	e.GET("/api/orders/:orderNumber", func(c echo.Context) error {
		orderNumber := c.Param("orderNumber")

		status := []string{"REGISTERED", "INVALID", "PROCESSING", "PROCESSED"}
		randomStatus, _ := rand.Int(rand.Reader, big.NewInt(4))
		orderStatus := status[randomStatus.Int64()]
		randomAccrual, _ := rand.Int(rand.Reader, big.NewInt(1000))

		processedOrder := ProcessedOrder{
			OrderNumber: orderNumber,
			OrderStatus: orderStatus,
			Accrual: func() decimal.Decimal {
				if orderStatus != "PROCESSED" {
					return decimal.Zero
				}
				return decimal.NewFromInt(randomAccrual.Int64())
			}(),
		}
		logger.Info("Processed order", zap.Any("order", processedOrder))

		return c.JSON(http.StatusOK, processedOrder)
	})

	errStart := e.Start(config.Address)
	if errStart != nil && !errors.Is(errStart, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type ProcessedOrder struct {
	OrderNumber string          `json:"order"`
	OrderStatus string          `json:"status"`
	Accrual     decimal.Decimal `json:"accrual"`
}
