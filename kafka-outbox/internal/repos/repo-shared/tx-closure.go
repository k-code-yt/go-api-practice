package reposhared

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func TxClosure[T any](ctx context.Context, repo *sqlx.DB, fn func(ctx context.Context, tx *sqlx.Tx) (T, error)) (T, error) {
	tx, err := repo.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		panic("unable to start TX")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}

		if err != nil {
			tx.Rollback()
			return
		}

		err = tx.Commit()
		if err != nil {
			fmt.Printf("err on commit = %v\n", err)
		}
	}()

	res, err := fn(ctx, tx)
	return res, err
}
