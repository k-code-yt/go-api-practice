package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
)

type PaymentCreatedEvent struct {
	ID          int    `json:"id"`
	OrderNumber string `json:"order_number"`
	Amount      string `json:"amount"`
	Status      string `json:"status"`
	CreatedAt   int    `json:"create_at"`
	UpdatedAt   int    `json:"updated_at"`
}

type InboxEventType string

const (
	InboxEventType_PaymentCreated InboxEventType = "payment_created"
)

type InboxEventStatus string

const (
	InboxEventStatus_Pending  InboxEventStatus = "pending"
	InboxEventStatus_Produced InboxEventStatus = "produced"
)

type InboxEventParentType string

const (
	InboxEventParentType_Payment InboxEventParentType = "payment"
)

type InboxEvent struct {
	ID              int              `db:"id" json:"EventId"`
	InboxEventType  string           `db:"type" json:"EventType"`
	OutboxEventId   string           `db:"outbox_event_id" json:"OutboxEventId"`
	OutboxCreatedAt time.Time        `db:"outbox_created_at" json:"Timespamp"`
	CreatedAt       time.Time        `db:"created_at" json:"CreatedAt"`
	Status          InboxEventStatus `db:"status" json:"Status"`

	ParentId       string               `db:"parent_id" json:"ParentId"`
	ParentType     InboxEventParentType `db:"parent_type" json:"ParentType"`
	ParentMetadata json.RawMessage      `db:"parent_metadata" json:"ParentMetadata"`
}

func NewInboxEvent(outboxEvent *Event) *InboxEvent {
	return &InboxEvent{
		InboxEventType:  outboxEvent.EventType,
		OutboxEventId:   outboxEvent.EventId,
		OutboxCreatedAt: outboxEvent.Timespamp,
		CreatedAt:       time.Now(),
		ParentId:        outboxEvent.ParentId,
		ParentType:      InboxEventParentType(outboxEvent.ParentType),
		ParentMetadata:  outboxEvent.ParentMetadata,
		Status:          InboxEventStatus(outboxEvent.Status),
	}
}

type InboxEventRepo struct {
	repo      *sqlx.DB
	tableName string
}

func NewInboxEventRepo(db *sqlx.DB) *InboxEventRepo {
	return &InboxEventRepo{
		repo:      db,
		tableName: "audit",
	}
}

func (r *InboxEventRepo) GetRepo() *sqlx.DB {
	return r.repo
}

func (r *InboxEventRepo) Insert(ctx context.Context, tx *sqlx.Tx, e *InboxEvent) (int, error) {
	query := fmt.Sprintf(`INSERT INTO %s (type, "outbox_event_id", "outbox_created_at", "created_at", parent_id, parent_metadata, status, parent_type) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`, r.tableName)
	rows, err := tx.QueryxContext(ctx, query, e.InboxEventType, e.OutboxEventId, e.OutboxCreatedAt, e.CreatedAt, e.ParentId, e.ParentMetadata, e.Status, e.ParentType)
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
