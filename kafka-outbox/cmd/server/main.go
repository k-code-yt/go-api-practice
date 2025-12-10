package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo"
	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/payment"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/service/payment"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/shared"
)

type Server struct {
	addr           string
	msgCH          chan *shared.Message
	paymentService *payment.PaymentService
	exitCH         chan struct{}
}

func NewServer(addr string, db *sqlx.DB) *Server {
	ps := payment.NewPaymentService(db)
	return &Server{
		msgCH:          make(chan *shared.Message, 512),
		addr:           addr,
		paymentService: ps,
		exitCH:         make(chan struct{}),
	}
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	paym := paymentrepo.NewPayment("123", 123, "created")
	s.paymentService.Save(ctx, paym)
}

func main() {
	db, err := repo.NewDBConn()
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	s := NewServer(":7576", db)
	mux := http.NewServeMux()
	mux.HandleFunc("/payment", s.handleCreatePayment)
	log.Fatal(http.ListenAndServe(s.addr, nil))
}
