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
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/db/postgres"
	repo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/service"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-outbox/pkg/types"
)

type Server struct {
	addr           string
	msgCH          chan *pkgtypes.Message[*repo.Event]
	paymentService *service.PaymentService
	outboxService  *service.OutboxService
	exitCH         chan struct{}
}

func NewServer(addr string, db *sqlx.DB) *Server {
	eventRepo := repo.NewEventRepo(db)
	pr := repo.NewPaymentRepo(db)
	ps := service.NewPaymentService(pr, eventRepo)
	os := service.NewOutbox(eventRepo)
	return &Server{
		msgCH:  make(chan *pkgtypes.Message[*repo.Event], 64),
		exitCH: make(chan struct{}),
		addr:   addr,

		paymentService: ps,
		outboxService:  os,
	}
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	paym := repo.NewPayment("123", 123, "created")
	s.paymentService.Save(ctx, paym)
}

func (s *Server) simulateCreatePayment() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	amount := rand.Intn(1000)
	orderN := strconv.Itoa(amount)
	paym := repo.NewPayment(orderN, amount, "created")
	s.paymentService.Save(ctx, paym)
}

func main() {
	db, err := dbpostgres.NewDBConn()
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
