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
	Timespamp time.Time `db:"created"`
}

func NewEvent() *Event {
	id := GenerateRandomString(9)
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

func (r *EventRepo) Insert(ctx context.Context, e *Event) (string, error) {
	tx, err := r.repo.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	defer func() {
		err := tx.Rollback()
		if err != nil {
			fmt.Printf("err rolling back = %v\n", err)
		}
	}()

	if err != nil {
		panic(fmt.Sprintf("unable to start TX, err = %v\n", err))
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (event_name, event_id, created) VALUES($1, $2, $3)", r.tableName), e.EventId, e.EventName, e.Timespamp)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		fmt.Printf("err on commit = %v\n", err)
		return "", err
	}
	return e.EventId, nil
}

func (r *EventRepo) Get(ctx context.Context, eventID string) *Event {
	tx, err := r.repo.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	defer func() {
		err := tx.Rollback()
		if err != nil {
			fmt.Printf("err rolling back = %v\n", err)
		}
	}()

	if err != nil {
		panic(fmt.Sprintf("unable to start TX, err = %v\n", err))
	}

	row := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT event_id from %s WHERE event_id = $1", r.tableName), eventID)
	e := &Event{}
	row.Scan(e)

	err = tx.Commit()
	if err != nil {
		fmt.Printf("err on commit = %v\n", err)
		return nil
	}

	if e.EventId == "" {
		return nil
	}
	return e
}
