package dbpostgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/config"
)

type DBPostgresOptions struct {
	DBname string
}

func getDBConnString(opts *DBPostgresOptions) string {
	if opts.DBname == "" {
		return config.GetDefaultConnString()
	}
	return config.GetConnString(opts.DBname)
}

func NewDBConn(opts *DBPostgresOptions) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", getDBConnString(opts))
	if err != nil {
		return nil, err
	}
	return db, nil
}
