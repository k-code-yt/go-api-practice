package main

import (
	"fmt"

	"github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/ports"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
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

func (svc *InvoiceService) SensorDataStream(id string, lat, lng float64) error {
	logrus.WithFields(logrus.Fields{
		"ID": id,
	}).Info("Received GRPC steam data into SVC")

	// randVal := rand.Intn(10)
	// if randVal > 5 {
	// 	return fmt.Errorf("generating random err in GRPC stream for ID = %s", id)
	// }
	return nil
}

func NewInvoiceService() ports.Invoicer {
	var invSL []*shared.Invoice
	return &InvoiceService{
		invoicesSL: invSL,
	}
}
