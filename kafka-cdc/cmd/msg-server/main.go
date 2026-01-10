package main

import (
	"context"
	"fmt"
	"time"

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
	service  *service.InboxService
}

func NewServer(inboxRepo *repo.InboxEventRepo) *Server {
	s := service.NewInboxService(inboxRepo)
	return &Server{
		service: s,
	}
}

func (s *Server) addConsumer(handlerResigry *handlers.Registry) *Server {
	c := consumer.NewKafkaConsumer([]string{pkgconstants.DebDefaultTopic}, handlerResigry)
	s.consumer = c
	s.service.AddConsumer(c)
	return s
}

func (s *Server) handleMsg(paymentEvent *debezium.DebeziumMessage[domain.Payment]) {
	<-s.consumer.ReadyCH
	// TODO -> share context w/ handler
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	inboxEvent, err := service.PaymentToInbox(paymentEvent)
	if err != nil {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
		return
	}

	inboxID, err := s.service.Save(ctx, inboxEvent, paymentEvent.Metadata)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inboxEvent.AggregateId,
				"OFFSET":      paymentEvent.Metadata.Offset,
				"PRTN":        paymentEvent.Metadata.Partition,
			},
		).Error("INSERT:ERROR")
		return
	}
}

func main() {
	dbOpts := &dbpostgres.DBPostgresOptions{}
	dbOpts.DBname = pkgconstants.DBName
	db, err := dbpostgres.NewDBConn(dbOpts)
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	inboxRepo := repo.NewInboxEventRepo(db)
	s := NewServer(inboxRepo)
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
