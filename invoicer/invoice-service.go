package main

import (
	"github.com/k-code-yt/go-api-practice/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/shared"
)

const (
	Invoice_DefaultRate = 2
)

type InvoiceService struct {
}

func (svc *InvoiceService) SaveInvoice(d *shared.Distance) *shared.Invoice {
	amount := d.Value * Invoice_DefaultRate
	return shared.NewInvoice(amount, shared.CategoryDistance)
}

func NewInvoiceService() ports.Invoicer {
	return &InvoiceService{}
}
