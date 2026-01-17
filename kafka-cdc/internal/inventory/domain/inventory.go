package domain

import (
	"crypto/rand"
	"math"
	"time"
)

type Inventory struct {
	ID          int       `db:"id"`
	ProductName string    `db:"product_name"`
	Status      string    `db:"status"`
	Quantity    int       `db:"quantity"`
	LastUpdated time.Time `db:"last_updated"`

	OrderNumber string `db:"order_number"`
	PaymentId   string `db:"payment_id"`
}

func NewInventoryReservation(orderNumber string, paymentID string, amount int) *Inventory {
	pricePerProduct := 10
	qty := int(math.Floor(float64(amount / pricePerProduct)))
	pName := rand.Text()[:9]
	return &Inventory{
		ProductName: pName,
		Status:      "reserved",
		Quantity:    qty,
		LastUpdated: time.Now(),

		OrderNumber: orderNumber,
		PaymentId:   paymentID,
	}
}
