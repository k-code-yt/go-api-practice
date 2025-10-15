package client

import (
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
)

type TransportClient interface {
	SaveInvoice(distance shared.Distance) error
	GetInvoice(id string) (*shared.Invoice, error)
	AcceptWSLoop()
	ReadServerLoop()
}
