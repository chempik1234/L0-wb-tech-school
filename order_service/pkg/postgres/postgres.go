package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required import
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required import
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config from postgres package is supposed to be used with an env-prefix of "POSTGRES_"
type Config struct {
	Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port     uint16 `yaml:"port" env:"PORT" env-default:"5432"`
	Username string `yaml:"user" env:"USER" env-default:"postgres"`
	Password string `yaml:"password" env:"PASSWORD" env-default:"1234"`
	Database string `yaml:"db" env:"DB" env-default:"postgres"`

	MaxConns int32 `yaml:"max_conn" env:"MAX_CONN" env-default:"10"`
	MinConns int32 `yaml:"min_conn" env:"MIN_CONN" env-default:"5"`
}

// New creates a new postgres pool with given settings
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
		return nil, fmt.Errorf("unable to connect to postgres by link %s: %v", connString, err)
	}

	m, err := migrate.New(
		"file:///app/db/migrations",
		connStringShort,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create migrations table by link %s: %v", connStringShort, err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("unable to apply migrations: %v", err)
	}

	return conn, nil
}
