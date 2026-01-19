package domain

import "time"

type InventoryCreatedEvent struct {
	ID          int       `db:"id"`
	ProductName string    `db:"product_name"`
	Status      string    `db:"status"`
	Quantity    int       `db:"quantity"`
	LastUpdated time.Time `db:"last_updated"`

	OrderNumber string `db:"order_number"`
	PaymentId   string `db:"payment_id"`
}
