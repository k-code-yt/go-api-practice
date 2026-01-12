package main

import (
	"fmt"

	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer/handlers"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/service"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	"github.com/sirupsen/logrus"
)

type Server struct {
	consumer *consumer.KafkaConsumer
	service  *service.InventoryService
}

func NewServer(inboxRepo *repo.InboxEventRepo, invRepo *repo.InventoryRepo) *Server {
	err := debezium.RegisterConnector("http://localhost:8083", pkgconstants.DebInventoryDBConnectorName, pkgconstants.DBNameInventory, fmt.Sprintf("public.%s", pkgconstants.DBTableName_Inventory))
	if err != nil {
		fmt.Printf("err on deb-m conn %v\n", err)
		panic(err)
	}

	s := service.NewInventoryService(inboxRepo, invRepo)
	return &Server{
		service: s,
	}
}

func (s *Server) addConsumer(handlerResigry *handlers.Registry) *Server {
	c := consumer.NewKafkaConsumer([]string{pkgconstants.DebPaymentTopic}, handlerResigry)
	s.consumer = c
	s.service.AddConsumer(c)
	return s
}

func (s *Server) handleMsg(paymentEvent *debezium.DebeziumMessage[domain.Payment]) {
	<-s.consumer.ReadyCH

	inv, inbox, err := service.PaymentToInventory(paymentEvent)
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

func main() {
	dbOpts := &dbpostgres.DBPostgresOptions{}
	dbOpts.DBname = pkgconstants.DBNameInventory
	db, err := dbpostgres.NewDBConn(dbOpts)
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	inboxRepo := repo.NewInboxEventRepo(db)
	invRepo := repo.NewInventoryRepo(db)
	s := NewServer(inboxRepo, invRepo)
	registry := handlers.NewRegistry()
	paymentCreatedHandler := handlers.NewPaymentCreatedHandler()

	registry.AddHandler(paymentCreatedHandler.Handler, pkgconstants.EventType_PaymentCreated)

	s.addConsumer(registry)

	go func() {
		for msg := range paymentCreatedHandler.MsgCH {
			go s.handleMsg(msg)
		}
	}()

	s.consumer.RunConsumer()
}
