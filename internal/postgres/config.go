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

func (config Config) ConnectionURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", config.Username, config.Password, config.Host, config.Port, config.Database)
}

func migrations() Config {
	return Config{
		Username:      os.Getenv("PG_MIGRATIONS_USER"),
		Password:      os.Getenv("PG_MIGRATIONS_PASSWORD"),
		Host:          os.Getenv("PG_HOST"),
		Port:          os.Getenv("PG_PORT"),
		Database:      os.Getenv("PG_DATABASE"),
		Schema:        os.Getenv("PG_SCHEMA"),
		MigrationsDir: os.Getenv("PG_MIGRATIONS_DIR"),
	}
}

func usage() Config {
	return Config{
		Username: os.Getenv("PG_USERNAME"),
		Password: os.Getenv("PG_PASSWORD"),
		Host:     os.Getenv("PG_HOST"),
		Port:     os.Getenv("PG_PORT"),
		Database: os.Getenv("PG_DATABASE"),
		Schema:   os.Getenv("PG_SCHEMA"),
	}
}
