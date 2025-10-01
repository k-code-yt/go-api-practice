package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/k-code-yt/go-api-practice/shared"

	"github.com/sirupsen/logrus"
)

const (
	Invoice_DefaultRate = 2
)

type Invoicer interface {
	SaveInvoice(d *shared.Distance) *shared.Invoice
}

type InvoiceService struct {
}

func (svc *InvoiceService) SaveInvoice(d *shared.Distance) *shared.Invoice {
	amount := d.Value * Invoice_DefaultRate
	return shared.NewInvoice(amount, shared.CategoryDistance)
}

func SaveInvoice(svc Invoicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"method":  r.Method,
			"service": "Invoicer",
		}).Info("received save request")
		switch r.Method {
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid payload"})
			}
			defer r.Body.Close()
			var d shared.Distance
			if err = json.Unmarshal(body, &d); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid payload"})
			}
			invoice := svc.SaveInvoice(&d)
			logrus.WithFields(logrus.Fields{
				"invoice": invoice,
			}).Info("saved invoice SUCCESS")
			writeJSON(w, http.StatusOK, invoice)
		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"message": fmt.Sprintf("%s does not exist", r.Method)})
			return
		}

	}
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "applicaiton/json")
	return json.NewEncoder(rw).Encode(v)
}

func main() {
	svc := new(InvoiceService)

	http.HandleFunc("/invoice", SaveInvoice(svc))
	logrus.Infof("Listening on port %s\n", shared.HTTPPortInvoice)
	log.Fatal(http.ListenAndServe(shared.HTTPPortInvoice, nil))
}
