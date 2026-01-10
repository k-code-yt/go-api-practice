package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
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
	ID                 int                    `db:"id" json:"EventId"`
	InboxEventType     pkgconstants.EventType `db:"type" json:"EventType"`
	AggregateId        string                 `db:"aggregate_id" json:"AggregateEventId"`
	AggregateCreatedAt time.Time              `db:"aggregate_created_at" json:"Timespamp"`
	AggregateType      InboxEventParentType   `db:"aggregate_type" json:"AggregateType"`
	AggregateMetadata  json.RawMessage        `db:"aggregate_metadata" json:"AggregateMetadata"`
	CreatedAt          time.Time              `db:"created_at" json:"CreatedAt"`
	Status             InboxEventStatus       `db:"status" json:"Status"`
}

type InboxEventRepo struct {
	repo      *sqlx.DB
	tableName string
}

func NewInboxEventRepo(db *sqlx.DB) *InboxEventRepo {
	return &InboxEventRepo{
		repo:      db,
		tableName: pkgconstants.DBTableName_InboxEvents,
	}
}

func (r *InboxEventRepo) GetRepo() *sqlx.DB {
	return r.repo
}

func (r *InboxEventRepo) Insert(ctx context.Context, tx *sqlx.Tx, e *InboxEvent) (int, error) {
	query := fmt.Sprintf(`INSERT INTO %s (type, "aggregate_id", "aggregate_created_at", "created_at", aggregate_metadata, status, aggregate_type) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`, r.tableName)
	rows, err := tx.QueryxContext(ctx, query, e.InboxEventType, e.AggregateId, e.AggregateCreatedAt, e.CreatedAt, e.AggregateMetadata, e.Status, e.AggregateType)
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
