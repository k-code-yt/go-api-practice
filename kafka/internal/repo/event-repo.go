package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Event struct {
	EventId   string    `db:"event_id"`
	EventName string    `db:"event_type"`
	Timespamp time.Time `db:"timestamp"`
}

func NewEvent() *Event {
	id := GenerateRandomString(15)
	return &Event{
		EventId:   id,
		EventName: "test_event",
		Timespamp: time.Now(),
	}
}

type EventRepo struct {
	repo      *sqlx.DB
	tableName string
}

func NewEventRepo(db *sqlx.DB) *EventRepo {
	return &EventRepo{
		repo:      db,
		tableName: "events",
	}
}

func (r *EventRepo) Insert(ctx context.Context, tx *sqlx.Tx, e *Event) (string, error) {
	_, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (event_name, event_id, timestamp) VALUES($1, $2, $3)", r.tableName), e.EventName, e.EventId, e.Timespamp)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err
	}
	return e.EventId, nil
}

func (r *EventRepo) Get(ctx context.Context, tx *sqlx.Tx, eventID string) *Event {
	e := &Event{}
	q := fmt.Sprintf("SELECT event_id from %s WHERE event_id = $1", r.tableName)
	err := tx.GetContext(ctx, e, q, eventID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
	}
	return e
}

func TxClosure[T any](ctx context.Context, r *EventRepo, fn func(ctx context.Context, tx *sqlx.Tx) (T, error)) (T, error) {
	tx, err := r.repo.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		panic("unable to start TX")
	}
	defer func() {
		tx.Rollback()
	}()

	res, err := fn(ctx, tx)
	if err != nil {
		return res, err
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("err on commit = %v\n", err)
		return res, err
	}
	return res, err
}
