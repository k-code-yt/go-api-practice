package main

import (
	"context"
	"fmt"
	"time"

	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer/handlers"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/service"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
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

func (s *Server) handleMsg(msg repo.PaymentCreatedEvent) {
	<-s.consumer.ReadyCH
	_, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	// _, err := s.service.Save(ctx, msg)
	// if err != nil {
	// 	fmt.Printf("ERR on DB SAVE = %v\n", err)
	// 	return
	// }
	// logrus.WithFields(logrus.Fields{
	// 	"EventID": msg.Data.EventId,
	// 	"Offset":  msg.Metadata.Offset,
	// 	"PRTN":    msg.Metadata.Partition,
	// }).Info("MSG:SAVED")
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

	registry.AddHandler(paymentCreatedHandler.Handler, repo.EventType_PaymentCreated)

	s.addConsumer(registry)

	go func() {
		for msg := range paymentCreatedHandler.MsgCH {
			go s.handleMsg(msg)
		}
	}()

	s.consumer.RunConsumer()
}
