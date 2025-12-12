package dbpostgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/config"
)

func getDBConnString() string {
	return config.GetDefaultConnString()
}

func NewDBConn() (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", getDBConnString())
	if err != nil {
		return nil, err
	}
	return db, nil
}
