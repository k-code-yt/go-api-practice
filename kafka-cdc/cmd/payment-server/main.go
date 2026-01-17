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
	payment "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/application"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/handlers"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/infra/msg"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/infra/repo"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
)

type Server struct {
	addr      string
	exitCH    chan struct{}
	consumer  *pkgkafka.KafkaConsumer
	msgRouter *handlers.MsgRouter
	// for testing
	paymentService *payment.PaymentService
}

func NewServer(addr string, db *sqlx.DB, DBName string) *Server {
	err := debezium.RegisterConnector("http://localhost:8083", debezium.GetDebPaymentDBConnectorName(DBName), DBName, fmt.Sprintf("public.%s", repo.DBTableName_Payment))
	if err != nil {
		fmt.Printf("err on deb-m conn %v\n", err)
		panic(err)
	}

	eventRepo := repo.NewEventRepo(db)
	pr := repo.NewPaymentRepo(db)
	ps := payment.NewPaymentService(pr, eventRepo)
	kafkaConsumer := pkgkafka.NewKafkaConsumer([]string{msg.DebInventoryTopic})

	msgRouter := handlers.NewMsgRouter(kafkaConsumer)
	invCreateHandler := handlers.NewInventoryCreatedHandler(ps)
	msgRouter.AddHandler(invCreateHandler.Handler, pkgconstants.EventType_InvetoryCreated)

	return &Server{
		exitCH:    make(chan struct{}),
		addr:      addr,
		msgRouter: msgRouter,

		paymentService: ps,
	}
}

// TODO -> remove from main
// FOR TESTING
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
	paym := domain.NewPayment(orderN, float64(amount), "reserved")
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

	s := NewServer(":7576", db, dbOpts.DBName)
	mux := http.NewServeMux()
	mux.HandleFunc("/payment", s.handleCreatePayment)

	// go func() {
	// 	ticker := time.NewTicker(time.Second * 15)
	// 	for range ticker.C {
	// 		s.createPayment()
	// 	}
	// }()

	log.Fatal(http.ListenAndServe(s.addr, nil))
}
