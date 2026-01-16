package postgres

import (
	"github.com/jmoiron/sqlx"
)

func getDBConnString(opts *PostgresConfig) string {
	if opts.DBName == "" {
		return GetDefaultConnString()
	}
	return GetConnString(opts)
}

func NewDBConn(opts *PostgresConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", getDBConnString(opts))
	if err != nil {
		return nil, err
	}
	return db, nil
}
