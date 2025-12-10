package repo

import (
	"github.com/jmoiron/sqlx"
)

func getDBConnString() string {
	return "host=localhost port=5432 user=user password=pass dbname=events sslmode=disable"
}

func NewDBConn() (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", getDBConnString())
	if err != nil {
		return nil, err
	}
	return db, nil
}
