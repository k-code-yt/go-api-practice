package avro

type CDCPayment struct {
	ID          int     `avro:"id"`
	OrderNumber string  `avro:"order_number"`
	Amount      string  `avro:"amount"`
	Status      *string `avro:"status"`
	CreatedAt   *int64  `avro:"created_at"`
	UpdatedAt   *int64  `avro:"updated_at"`
}
