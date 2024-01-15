package main

import (
	"os"
	"os/signal"
	"syscall"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/msmkdenis/yap-gophermart/docs" // docs is generated by Swag CLI, you have to import it.
	"github.com/msmkdenis/yap-gophermart/internal/app/gophermart"
)

//swag init --parseInternal --dir D:/GoProjects/GoPracticum/yap-gophermart/cmd/gophermart,D:/GoProjects/GoPracticum/yap-gophermart/internal --output D:/GoProjects/GoPracticum/yap-gophermart/docs/

// @title Swagger Gophermart API
// @version 1.0

// @host localhost:7000
func main() {
	quitSignal := make(chan os.Signal, 1)
	signal.Notify(quitSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	gophermart.Run(quitSignal)
}
