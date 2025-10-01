package client

import (
	"github.com/k-code-yt/go-api-practice/shared"
)

type TransportClient interface {
	SaveInvoice(distance shared.Distance) error
}
