package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
)

type PaymentRepo struct {
	repo      *sqlx.DB
	tableName string
}

func NewPaymentRepo(db *sqlx.DB) *PaymentRepo {
	return &PaymentRepo{
		repo:      db,
		tableName: "payment",
	}
}

func (r *PaymentRepo) Insert(ctx context.Context, tx *sqlx.Tx, p *domain.Payment) (int, error) {
	// ----
	// TODO -> investigate perfomance
	// v1 -> pretty, but overhead of prepare
	// ----

	// query := fmt.Sprintf("INSERT INTO %s (order_number, amount, status, created_at, updated_at) VALUES(:order_number, :amount, :status, :created_at, :updated_at) RETURNING id", r.tableName)
	// stmt, err := tx.PrepareNamedContext(ctx, query)
	// if err != nil {
	// 	return dbpostgres.NonExistingIntKey, err
	// }
	// defer stmt.Close()
	// err = stmt.Get(p, p)
	// if err != nil {
	// 	return dbpostgres.NonExistingIntKey, err
	// }
	// return p.ID, nil

	// v2 -> no prepare, better perf-ce
	query := fmt.Sprintf("INSERT INTO %s (order_number, amount, status, created_at, updated_at) VALUES($1, $2, $3, $4, $5) RETURNING id", r.tableName)
	rows, err := tx.QueryxContext(ctx, query, p.OrderNumber, p.Amount, p.Status, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		// fmt.Printf("err on insert = %v\n", err)
		return dbpostgres.NonExistingIntKey, err
	}
	defer rows.Close()
	var id int
	for rows.Next() {
		ids, err := rows.SliceScan()
		if err != nil {
			fmt.Printf("unable to get last insetID = %v\n", err)
			return dbpostgres.NonExistingIntKey, err
		}
		id = int(ids[0].(int64))
	}
	return id, nil
}

func (r *PaymentRepo) Get(ctx context.Context, tx *sqlx.Tx, ID int) *domain.Payment {
	e := &domain.Payment{}
	q := fmt.Sprintf("SELECT id from %s WHERE id = $1", r.tableName)
	err := tx.GetContext(ctx, e, q, ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
	}
	return e
}
