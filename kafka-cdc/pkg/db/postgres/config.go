package postgres

import (
	"fmt"
	"os"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func NewPostgresConfig(fallbackDBName string) *PostgresConfig {
	var postgres PostgresConfig

	postgres.Host = getEnv("POSTGRES_HOSTS", "localhost")
	postgres.Port = getEnv("POSTGRES_PORT", "5452")
	postgres.User = getEnv("POSTGRES_USER", "user")
	postgres.Password = getEnv("POSTGRES_PASSWORD", "pass")
	postgres.DBName = getEnv("POSTGRES_DATABASE", fallbackDBName)
	// postgres.SSL = getEnv("POSTGRES_SSL", falseStr) == trueStr
	// postgres.MigrationPath = getEnv("POSTGRES_MIGRATION_PATH", "")

	return &postgres

}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func GetDefaultConnString() string {
	return "host=localhost port=5452 user=user password=pass dbname=events sslmode=disable"
}

func GetConnString(options *PostgresConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", options.Host, options.Port, options.User, options.Password, options.DBName)
}
