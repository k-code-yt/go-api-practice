package main

import (
	"fmt"

	"github.com/k-code-yt/go-api-practice/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/shared"
)

const (
	Invoice_DefaultRate = 2
)

type InvoiceService struct {
	invoicesSL []*shared.Invoice
}

func (svc *InvoiceService) SaveInvoice(d *shared.Distance) *shared.Invoice {
	amount := d.Value * Invoice_DefaultRate
	inv := shared.NewInvoice(amount, shared.CategoryDistance)
	svc.invoicesSL = append(svc.invoicesSL, inv)
	return inv
}

func (svc *InvoiceService) GetInvoice(id string) (*shared.Invoice, error) {
	for _, v := range svc.invoicesSL {
		if v.ID == id {
			return v, nil

		}
	}
	return nil, fmt.Errorf("invoice with ID: %s does not exists", id)
}

func NewInvoiceService() ports.Invoicer {
	var invSL []*shared.Invoice
	return &InvoiceService{
		invoicesSL: invSL,
	}
}
