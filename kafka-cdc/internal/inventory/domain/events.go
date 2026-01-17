package domain

import "time"

type PaymentCreatedEvent struct {
	ID          int       `db:"id"`
	OrderNumber string    `db:"order_number"`
	Amount      float64   `db:"amount"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
