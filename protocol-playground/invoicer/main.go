package main

import (
	"log"
	"os"

	"github.com/k-code-yt/go-api-practice/protocol-playground/invoicer/transport"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
)

func main() {
	svc := NewInvoiceService()
	server := transport.NewServerTransport(shared.Invoicer_DefaultTransportType)
	err := server.Listen(svc)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

}
