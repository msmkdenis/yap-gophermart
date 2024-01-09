package db

import (
	"embed"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/msmkdenis/yap-gophermart/internal/apperrors"
	"github.com/msmkdenis/yap-gophermart/internal/utils"
)

//go:embed migration/*.sql
var migrationsFS embed.FS

type Migrations struct {
	migrations *migrate.Migrate
	logger     *zap.Logger
}

func NewMigrations(connection string, logger *zap.Logger) (*Migrations, error) {
	dbConfig, err := pgxpool.ParseConfig(connection)
	if err != nil {
		return nil, apperrors.NewValueError("Unable to parse connection string", utils.Caller(), err)
	}

	logger.Info(fmt.Sprintf("Connection %s", connection))

	dbURL := dbURL(dbConfig, sslMode(connection))

	driver, err := iofs.New(migrationsFS, "migration")
	if err != nil {
		return nil, apperrors.NewValueError("Unable to create iofs driver", utils.Caller(), err)
	}

	logger.Info(fmt.Sprintf("Connection to database %s", dbURL))

	migrations, err := migrate.NewWithSourceInstance("iofs", driver, dbURL)
	if err != nil {
		return nil, apperrors.NewValueError("Unable to create new migrations", utils.Caller(), err)
	}

	return &Migrations{
		migrations: migrations,
		logger:     logger,
	}, nil
}

func (m *Migrations) MigrateUp() error {
	err := m.migrations.Up()
	if err != nil && err.Error() != "no change" {
		return apperrors.NewValueError("Unable to up migrations", utils.Caller(), err)
	}
	return nil
}

func dbURL(config *pgxpool.Config, sslMode string) string {
	var dbURL strings.Builder

	dbURL.WriteString("postgres://")
	dbURL.WriteString(config.ConnConfig.User)
	dbURL.WriteString(":")
	dbURL.WriteString(config.ConnConfig.Password)
	dbURL.WriteString("@")
	dbURL.WriteString(config.ConnConfig.Host)
	dbURL.WriteString(":")
	dbURL.WriteString(fmt.Sprint(config.ConnConfig.Port))
	dbURL.WriteString("/")
	dbURL.WriteString(config.ConnConfig.Database)
	dbURL.WriteString("?sslmode=")
	if config.ConnConfig.TLSConfig == nil {
		dbURL.WriteString("disable")
	} else {
		dbURL.WriteString(sslMode)
	}

	return dbURL.String()
}

func sslMode(connection string) string {
	con := strings.Split(connection, " ")
	sslMode := ""
	for _, v := range con {
		pair := strings.Split(v, "=")
		if pair[0] == "sslmode" {
			sslMode = pair[1]
		}
	}

	return sslMode
}
