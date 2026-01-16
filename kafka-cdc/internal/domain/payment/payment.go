package payment

import "time"

type Payment struct {
	ID          int       `db:"id"`
	OrderNumber string    `db:"order_number"`
	Amount      float64   `db:"amount"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func NewPayment(orderNumber string, amount float64, status string) *Payment {
	return &Payment{
		OrderNumber: orderNumber,
		Amount:      amount,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
