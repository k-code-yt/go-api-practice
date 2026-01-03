package reposhared

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func TxClosure[T any](ctx context.Context, repo *sqlx.DB, fn func(ctx context.Context, tx *sqlx.Tx) (T, error)) (res T, err error) {
	tx, err := repo.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return res, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}

		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("tx failed: %w, rollback failed: %v", err, rbErr)
			}
			return
		}

		err = tx.Commit()
		if err != nil {
			err = fmt.Errorf("failed to commit transaction: %w", err)
		}
	}()

	res, err = fn(ctx, tx)
	return res, err
}
