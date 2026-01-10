package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgutils "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/utils"
	_ "github.com/lib/pq"
)

type EventStatus string

const (
	EventStatus_Pending  EventStatus = "pending"
	EventStatus_Produced EventStatus = "produced"
)

type EventParentType string

const (
	EventParentType_Payment EventParentType = "payment"
)

type Event struct {
	EventId   string      `db:"event_id" json:"EventId"`
	EventType string      `db:"event_type" json:"EventType"`
	Timespamp time.Time   `db:"timestamp" json:"Timespamp"`
	Status    EventStatus `db:"status" json:"Status"`

	ParentId       string          `db:"parent_id" json:"ParentId"`
	ParentType     EventParentType `db:"parent_type" json:"ParentType"`
	ParentMetadata json.RawMessage `db:"parent_metadata" json:"ParentMetadata"`
}

func NewEvent(eventName pkgconstants.EventType, parentId string, parentType EventParentType, parentMetadata json.RawMessage) *Event {
	id := pkgutils.GenerateRandomString(15)
	return &Event{
		EventId:        id,
		EventType:      string(eventName),
		Timespamp:      time.Now(),
		ParentType:     parentType,
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
		tableName: pkgconstants.DBTableName_OutboxEvents,
	}
}

func (r *EventRepo) GetRepo() *sqlx.DB {
	return r.repo
}

func (r *EventRepo) Insert(ctx context.Context, tx *sqlx.Tx, e *Event) (string, error) {
	_, err := tx.NamedExecContext(ctx, fmt.Sprintf("INSERT INTO %s (event_type, event_id, timestamp, parent_id, parent_metadata, status, parent_type) VALUES(:event_type, :event_id, :timestamp, :parent_id, :parent_metadata, :status, :parent_type)", r.tableName), e)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return "", err
	}
	return e.EventId, nil
}

func (r *EventRepo) UpdateStatusById(ctx context.Context, tx *sqlx.Tx, eventId string, status EventStatus) (string, error) {
	res, err := tx.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET status = $1 WHERE event_id = $2", r.tableName), status, eventId)
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

func (r *EventRepo) UpdateStatusByIds(ctx context.Context, tx *sqlx.Tx, eventIds []string, status EventStatus) (int, error) {
	if status == "" {
		status = EventStatus_Produced
	}
	query := fmt.Sprintf("UPDATE %s SET status = ? WHERE event_id IN (?)", r.tableName)
	query, args, err := sqlx.In(query, status, eventIds)
	if err != nil {
		return 0, err
	}

	query = tx.Rebind(query)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return 0, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return 0, err

	}

	return int(rows), nil
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
	q := fmt.Sprintf("SELECT * FROM %s WHERE status = $1", r.tableName)

	rows, err := tx.QueryxContext(ctx, q, EventStatus_Pending)
	if err != nil {
		if err == sql.ErrNoRows {
			return events, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		event := &Event{}
		err = rows.StructScan(event)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}
