package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/service"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type Server struct {
	consumer *consumer.KafkaConsumer[repo.Event]
	msgCH    chan *pkgtypes.Message[repo.Event]
	service  *service.InboxService
}

func NewServer(inboxRepo *repo.InboxEventRepo) *Server {
	s := service.NewInboxService(inboxRepo)
	return &Server{
		msgCH:   make(chan *pkgtypes.Message[repo.Event], 64),
		service: s,
	}
}

func (s *Server) addConsumer() *Server {
	c := consumer.NewKafkaConsumer(s.msgCH, []string{pkgconstants.DebDefaultTopic})
	s.consumer = c
	s.service.AddConsumer(c)
	return s
}

func (s *Server) handleMsg(msg *pkgtypes.Message[repo.Event]) {
	<-s.consumer.ReadyCH
	r := time.Duration(rand.IntN(5))
	time.Sleep(r * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	_, err := s.service.Save(ctx, msg)
	if err != nil {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
		return
	}
	logrus.WithFields(logrus.Fields{
		"EventID": msg.Data.EventId,
		"Offset":  msg.Metadata.Offset,
		"PRTN":    msg.Metadata.Partition,
	}).Info("MSG:SAVED")
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
	s := NewServer(inboxRepo).addConsumer()

	go func() {
		for msg := range s.msgCH {
			go s.handleMsg(msg)
		}
	}()

	s.consumer.RunConsumer()
}
