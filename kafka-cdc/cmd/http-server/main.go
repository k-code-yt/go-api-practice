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
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/service"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type Server struct {
	addr           string
	msgCH          chan *pkgtypes.Message[*repo.Event]
	paymentService *service.PaymentService
	exitCH         chan struct{}
}

func NewServer(addr string, db *sqlx.DB) *Server {
	err := debezium.RegisterConnector("http://localhost:8083", "audit_cdc")
	if err != nil {
		fmt.Printf("err on deb-m conn %v\n", err)
		panic(err)
	}

	eventRepo := repo.NewEventRepo(db)
	pr := repo.NewPaymentRepo(db)
	ps := service.NewPaymentService(pr, eventRepo)
	return &Server{
		msgCH:  make(chan *pkgtypes.Message[*repo.Event], 64),
		exitCH: make(chan struct{}),
		addr:   addr,

		paymentService: ps,
	}
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	_, err := s.createPayment()
	if err != nil {
		fmt.Printf("err creating PMNT %v\n", err)
	}
}

func (s *Server) createPayment() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	amount := rand.Intn(100_000)
	orderN := strconv.Itoa(amount)
	paym := domain.NewPayment(orderN, float64(amount), "created")
	return s.paymentService.Save(ctx, paym)
}

func main() {
	dbOpts := new(dbpostgres.DBPostgresOptions)
	dbOpts.DBname = pkgconstants.DBName
	db, err := dbpostgres.NewDBConn(dbOpts)
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	s := NewServer(":7576", db)
	mux := http.NewServeMux()
	mux.HandleFunc("/payment", s.handleCreatePayment)

	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for range ticker.C {
			s.createPayment()
		}
	}()

	log.Fatal(http.ListenAndServe(s.addr, nil))
}
