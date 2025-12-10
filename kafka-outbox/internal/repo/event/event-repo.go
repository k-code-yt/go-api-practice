package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/repo-shared"
	_ "github.com/lib/pq"
)

type EventType string

const (
	EventType_PaymentCreated EventType = "payment_created"
)

type EventStatus string

const (
	EventStatus_Pending  EventStatus = "pending"
	EventStatus_Produced EventStatus = "produced"
)

type Event struct {
	EventId   string      `db:"event_id"`
	EventType string      `db:"event_type"`
	Timespamp time.Time   `db:"timestamp"`
	Status    EventStatus `db:"status"`

	ParentId       string          `db:"parent_id"`
	ParentMetadata json.RawMessage `db:"parent_metadata"`
}

func NewEvent(eventName EventType, parentId string, parentMetadata json.RawMessage) *Event {
	id := reposhared.GenerateRandomString(15)
	return &Event{
		EventId:        id,
		EventType:      string(eventName),
		Timespamp:      time.Now(),
		ParentId:       parentId,
		ParentMetadata: parentMetadata,
		Status:         EventStatus_Pending,
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
	_, err := tx.NamedExecContext(ctx, fmt.Sprintf("INSERT INTO %s (event_type, event_id, timestamp, parent_id, parent_metadata, status) VALUES(:event_type, :event_id, :timestamp, :parent_id, :parent_metadata, :status)", r.tableName), e)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err
	}
	return e.EventId, nil
}

func (r *EventRepo) UpdateStatus(ctx context.Context, tx *sqlx.Tx, eventId string, status EventStatus) (string, error) {
	res, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET status = :status WHERE event_id = :id", r.tableName), status, eventId)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err

	}

	return strconv.Itoa(int(rows)), nil
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

func (r *EventRepo) GetAllPending(ctx context.Context, tx *sqlx.Tx) ([]*Event, error) {
	events := []*Event{}
	q := fmt.Sprintf("SELECT * from %s WHERE status = $1", r.tableName)
	rows, err := tx.QueryxContext(ctx, q, EventStatus_Pending)

	if err != nil {
		if err == sql.ErrNoRows {
			return events, nil
		}
		return nil, err
	}

	err = rows.Rows.Scan(events)
	if err != nil {
		return nil, err
	}

	return events, nil
}
