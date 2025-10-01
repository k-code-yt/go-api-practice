package ports

import "github.com/k-code-yt/go-api-practice/shared"

type Invoicer interface {
	SaveInvoice(d *shared.Distance) *shared.Invoice
}
