package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/application/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/inventory"
	paymdomain "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure"
	invrepo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory/constants"
	outboxrepo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/outbox"
	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer/handlers"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type Server struct {
	addr           string
	msgCH          chan *pkgtypes.Message[*invrepo.InboxEvent]
	paymentService *payment.PaymentService
	exitCH         chan struct{}
	consumer       *consumer.KafkaConsumer
}

func NewServer(addr string, db *sqlx.DB) *Server {
	DBNamePrimary := ""
	err := debezium.RegisterConnector("http://localhost:8083", debezium.GetDebPaymentDBConnectorName(DBNamePrimary), DBNamePrimary, fmt.Sprintf("public.%s", constants.DBTableName_Inventory))
	if err != nil {
		fmt.Printf("err on deb-m conn %v\n", err)
		panic(err)
	}

	eventRepo := outboxrepo.NewEventRepo(db)
	pr := paymentrepo.NewPaymentRepo(db)
	ps := payment.NewPaymentService(pr, eventRepo)
	return &Server{
		msgCH:  make(chan *pkgtypes.Message[*invrepo.InboxEvent], 64),
		exitCH: make(chan struct{}),
		addr:   addr,

		paymentService: ps,
	}
}

func (s *Server) addConsumer(handlerResigry *handlers.Registry) *Server {
	c := consumer.NewKafkaConsumer([]string{infrastructure.DebInventoryTopic}, handlerResigry)
	s.consumer = c
	return s
}

// TODO -> remove from main
func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	_, err := s.createPayment()
	if err != nil {
		fmt.Printf("err creating PMNT %v\n", err)
	}
}

// TODO -> remove from main
func (s *Server) handleInventoryReplyMsg(msg *infrastructure.DebeziumMessage[inventory.Inventory]) error {
	err := s.paymentService.Confirm(msg.Ctx, msg.Payload.After.ID)
	if err != nil {
		return err
	}

	logrus.WithFields(
		logrus.Fields{
			"PAY_ID": msg.Payload.After.PaymentId,
			"ORDER#": msg.Payload.After.OrderNumber,
		},
	).Info("PAYMENT:UPDATED")
	return nil
}

func (s *Server) createPayment() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	amount := rand.Intn(100_000)
	orderN := strconv.Itoa(amount)
	paym := paymdomain.NewPayment(orderN, float64(amount), "reserved")
	return s.paymentService.Save(ctx, paym)
}

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}

	dir := filepath.Dir(filename)
	envPath := filepath.Join(dir, ".env")

	if err := godotenv.Load(envPath); err != nil {
		log.Printf("No .env file found at %s", envPath)
	}
}

func main() {
	dbOpts := postgres.NewPostgresConfig("kafka_primary")
	db, err := postgres.NewDBConn(dbOpts)
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
	invCretedHandler := handlers.NewInventoryCreatedHandler()

	registry.AddHandler(invCretedHandler.Handler, paymdomain.EventType_InvetoryCreated)

	s.addConsumer(registry)

	go func() {
		for msg := range invCretedHandler.MsgCH {
			go s.handleInventoryReplyMsg(msg)
		}
	}()
	go s.consumer.RunConsumer()
	log.Fatal(http.ListenAndServe(s.addr, nil))
}
