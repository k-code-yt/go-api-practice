package ports

import "github.com/k-code-yt/go-api-practice/protocol-playground/shared"

type Invoicer interface {
	SaveInvoice(d *shared.Distance) *shared.Invoice
	GetInvoice(id string) (*shared.Invoice, error)
	SensorDataStream(id string, lat, lng float64) error
}
