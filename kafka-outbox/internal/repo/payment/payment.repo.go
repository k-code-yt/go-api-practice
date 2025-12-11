package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/event"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/repo-shared"
)

type Payment struct {
	ID          int       `db:"id"`
	OrderNumber string    `db:"order_number"`
	Amount      int       `db:"amount"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func NewPayment(orderNumber string, amount int, status string) *Payment {
	return &Payment{
		OrderNumber: orderNumber,
		Amount:      amount,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

type PaymentRepo struct {
	repo      *sqlx.DB
	tableName string
	eventRepo *event.EventRepo
}

func NewPaymentRepo(db *sqlx.DB, eventRepo *event.EventRepo) *PaymentRepo {
	return &PaymentRepo{
		repo:      db,
		tableName: "payment_orders",
		eventRepo: eventRepo,
	}
}

func (pr *PaymentRepo) Save(ctx context.Context, p *Payment) (int, error) {
	return reposhared.TxClosure(ctx, pr.repo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		fmt.Printf("starting DB operation for order# = %s\n", p.OrderNumber)

		paymentID, err := pr.insert(ctx, tx, p)
		if err != nil {
			exists := reposhared.IsDuplicateKeyErr(err)
			if exists {
				eMsg := fmt.Sprintf("already exists paymentID = %d\n", paymentID)
				return reposhared.NonExistingIntKey, errors.New(eMsg)
			}
			return reposhared.NonExistingIntKey, err
		}
		p.ID = paymentID
		metadata, err := json.Marshal(p)
		if err != nil {
			return reposhared.NonExistingIntKey, err
		}

		event := event.NewEvent(event.EventType_PaymentCreated, strconv.Itoa(paymentID), event.EventParentType_Payment, metadata)
		eventID, err := pr.eventRepo.Insert(ctx, tx, event)

		if err != nil {
			exists := reposhared.IsDuplicateKeyErr(err)
			if exists {
				eMsg := fmt.Sprintf("already exists paymentID = %d, eventID = %s\n", paymentID, eventID)

				return reposhared.NonExistingIntKey, errors.New(eMsg)
			}
			return reposhared.NonExistingIntKey, err
		}
		return paymentID, nil
	})
}

func (r *PaymentRepo) insert(ctx context.Context, tx *sqlx.Tx, p *Payment) (int, error) {
	query := fmt.Sprintf("INSERT INTO %s (order_number, amount, status, created_at, updated_at) VALUES($1, $2, $3, $4, $5) RETURNING id", r.tableName)

	rows, err := tx.QueryxContext(ctx, query, p.OrderNumber, p.Amount, p.Status, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		fmt.Printf("err on insert = %v\n", err)
		return reposhared.NonExistingIntKey, err
	}
	defer rows.Close()
	var id int
	for rows.Next() {
		ids, err := rows.SliceScan()
		if err != nil {
			fmt.Printf("unable to get last insetID = %v\n", err)
			return reposhared.NonExistingIntKey, err
		}
		id = int(ids[0].(int64))
	}
	return id, nil
}

func (r *PaymentRepo) Get(ctx context.Context, tx *sqlx.Tx, ID int) *Payment {
	e := &Payment{}
	q := fmt.Sprintf("SELECT id from %s WHERE id = $1", r.tableName)
	err := tx.GetContext(ctx, e, q, ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
	}
	return e
}
