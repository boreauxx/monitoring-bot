package postgres

import (
	"fmt"
	"os"
)

type Config struct {
	Username      string
	Password      string
	Host          string
	Port          string
	Database      string
	Schema        string
	MigrationsDir string
}

func (config Config) DbUrl() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", config.Username, config.Password, config.Host, config.Port, config.Database)
}

func Default() Config {
	return Config{
		Username:      os.Getenv("PG_USERNAME"),
		Password:      os.Getenv("PG_PASSWORD"),
		Host:          os.Getenv("PG_HOST"),
		Port:          os.Getenv("PG_PORT"),
		Database:      os.Getenv("PG_DATABASE"),
		Schema:        os.Getenv("PG_SCHEMA"),
		MigrationsDir: os.Getenv("MIGRATIONS_DIR"),
	}
}
