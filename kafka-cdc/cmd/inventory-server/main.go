package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/application/inventory"
	invetorydomain "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/inventory"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory/constants"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer/handlers"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	"github.com/sirupsen/logrus"
)

type Server struct {
	consumer *consumer.KafkaConsumer
	service  *inventory.InventoryService
}

func NewServer(inboxRepo *repo.InboxEventRepo, invRepo *repo.InventoryRepo) *Server {
	DBNameInventory := ""
	err := debezium.RegisterConnector("http://localhost:8083", debezium.GetDebPaymentDBConnectorName(DBNameInventory), DBNameInventory, fmt.Sprintf("public.%s", constants.DBTableName_Inventory))
	if err != nil {
		fmt.Printf("err on deb-m conn %v\n", err)
		panic(err)
	}

	s := inventory.NewInventoryService(inboxRepo, invRepo)
	return &Server{
		service: s,
	}
}

func (s *Server) addConsumer(handlerResigry *handlers.Registry) *Server {
	c := consumer.NewKafkaConsumer([]string{infrastructure.DebPaymentTopic}, handlerResigry)
	s.consumer = c
	s.service.AddConsumer(c)
	return s
}

func (s *Server) handleMsg(paymentEvent *infrastructure.DebeziumMessage[payment.Payment]) {
	<-s.consumer.ReadyCH

	inv, inbox, err := inventory.PaymentToInventory(paymentEvent)
	if err != nil {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
		return
	}

	inboxID, err := s.service.Save(paymentEvent.Ctx, inbox, inv, paymentEvent.Metadata)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inbox.AggregateId,
				"OFFSET":      paymentEvent.Metadata.Offset,
				"PRTN":        paymentEvent.Metadata.Partition,
			},
		).Error("INSERT:ERROR")
		return
	}
	logrus.WithFields(
		logrus.Fields{
			"inboxID":     inboxID,
			"aggregateID": inbox.AggregateId,
			"OFFSET":      paymentEvent.Metadata.Offset,
			"PRTN":        paymentEvent.Metadata.Partition,
		},
	).Info("INSERT:SUCCESS")
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
	dbOpts := postgres.NewPostgresConfig("kafka_inventory")
	db, err := postgres.NewDBConn(dbOpts)
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	inboxRepo := repo.NewInboxEventRepo(db)
	invRepo := repo.NewInventoryRepo(db)
	s := NewServer(inboxRepo, invRepo)
	registry := handlers.NewRegistry()
	paymentCreatedHandler := handlers.NewPaymentCreatedHandler()

	registry.AddHandler(paymentCreatedHandler.Handler, invetorydomain.EventType_PaymentCreated)

	s.addConsumer(registry)

	go func() {
		for msg := range paymentCreatedHandler.MsgCH {
			go s.handleMsg(msg)
		}
	}()

	s.consumer.RunConsumer()
}
