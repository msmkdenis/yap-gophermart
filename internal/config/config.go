package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	Address              string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	Secret               string `env:"SECRET"`
	TokenName            string `env:"TOKEN_NAME"`
}

func NewConfig() *Config {
	config := &Config{}

	flag.StringVar(&config.Address, "a", "localhost:7000", "Адрес и порт запуска сервиса")
	flag.StringVar(&config.DatabaseURI, "d", "user=postgres password=postgres host=localhost database=yap-gophermart sslmode=disable", "Адрес подключения к базе данных")
	flag.StringVar(&config.AccrualSystemAddress, "r", "http://localhost:8080", "Адрес подключения к базе данных")
	flag.StringVar(&config.Secret, "s", "supersecretkey", "Секрет для JWT")
	flag.StringVar(&config.TokenName, "t", "token", "Enter token name Or use TOKEN_NAME env")

	if err := env.Parse(config); err != nil {
		fmt.Printf("%+v\n", err)
	}

	return config
}
