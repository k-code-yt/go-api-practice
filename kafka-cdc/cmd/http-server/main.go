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
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer/handlers"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/service"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type Server struct {
	addr           string
	msgCH          chan *pkgtypes.Message[*repo.InboxEvent]
	paymentService *service.PaymentService
	exitCH         chan struct{}
	consumer       *consumer.KafkaConsumer
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
		msgCH:  make(chan *pkgtypes.Message[*repo.InboxEvent], 64),
		exitCH: make(chan struct{}),
		addr:   addr,

		paymentService: ps,
	}
}

func (s *Server) addConsumer(handlerResigry *handlers.Registry) *Server {
	c := consumer.NewKafkaConsumer([]string{pkgconstants.DebInboxTopic}, handlerResigry)
	s.consumer = c
	return s
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	_, err := s.createPayment()
	if err != nil {
		fmt.Printf("err creating PMNT %v\n", err)
	}
}

func (s *Server) handleInboxReplyMsg(msg *repo.InboxEvent) {
	logrus.WithFields(
		logrus.Fields{
			"ID":   msg.AggregateId,
			"TYPE": msg.AggregateType,
		},
	).Info("INBOX_REPLY:SUCCESS")
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

	registry := handlers.NewRegistry()
	inboxCretedHandler := handlers.NewInboxReplyCreatedHandler()

	registry.AddHandler(inboxCretedHandler.Handler, pkgconstants.EventType_InboxCreated)

	s.addConsumer(registry)

	go func() {
		for msg := range inboxCretedHandler.MsgCH {
			go s.handleInboxReplyMsg(&msg.Payload.After)
		}
	}()
	go s.consumer.RunConsumer()
	log.Fatal(http.ListenAndServe(s.addr, nil))
}
