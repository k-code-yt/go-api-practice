package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/event"
	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/payment"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/service/outbox"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/service/payment"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/shared"
)

type Server struct {
	addr           string
	msgCH          chan *shared.Message
	paymentService *payment.PaymentService
	outboxService  *outbox.OutboxService
	exitCH         chan struct{}
}

func NewServer(addr string, db *sqlx.DB) *Server {
	eventRepo := event.NewEventRepo(db)
	pr := paymentrepo.NewPaymentRepo(db, eventRepo)
	ps := payment.NewPaymentService(pr)
	os := outbox.NewOutbox(eventRepo)
	return &Server{
		msgCH:  make(chan *shared.Message, 512),
		exitCH: make(chan struct{}),
		addr:   addr,

		paymentService: ps,
		outboxService:  os,
	}
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	paym := paymentrepo.NewPayment("123", 123, "created")
	s.paymentService.Save(ctx, paym)
}

func (s *Server) simulateCreatePayment() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	amount := rand.Intn(1000)
	orderN := strconv.Itoa(amount)
	paym := paymentrepo.NewPayment(orderN, amount, "created")
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

	// go func() {
	// 	ticker := time.NewTicker(time.Second * 2)
	// 	for range ticker.C {
	// 		s.simulateCreatePayment()
	// 	}
	// }()

	log.Fatal(http.ListenAndServe(s.addr, nil))
}
