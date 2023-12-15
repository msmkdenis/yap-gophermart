package main

import (
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/msmkdenis/yap-gophermart/internal/app"
)

func main() {
	app.GophermartRun()
}
