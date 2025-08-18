package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string `yaml:"POSTGRES_HOST" env:"POSTGRES_HOST" env-default:"localhost"`
	Port     uint16 `yaml:"POSTGRES_PORT" env:"POSTGRES_PORT" env-default:"5432"`
	Username string `yaml:"POSTGRES_USER" env:"POSTGRES_USER" env-default:"postgres"`
	Password string `yaml:"POSTGRES_PASSWORD" env:"POSTGRES_PASSWORD" env-default:"1234"`
	Database string `yaml:"POSTGRES_DB" env:"POSTGRES_DB" env-default:"postgres"`

	MaxConns int32 `yaml:"POSTGRES_MAX_CONN" env:"POSTGRES_MAX_CONN" env-default:"10"`
	MinConns int32 `yaml:"POSTGRES_MIN_CONN" env:"POSTGRES_MIN_CONN" env-default:"5"`
}

func New(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_min_conns=%d&pool_max_conns=%d",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.MinConns,
		config.MaxConns,
	)
	connStringShort := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	conn, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres: %v", err)
	}

	m, err := migrate.New(
		"file:///app/db/migrations",
		connStringShort,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create migrations table: %v", err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("unable to apply migrations: %v", err)
	}

	return conn, nil
}
