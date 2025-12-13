package main

import (
	"fmt"

	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/kafka/consumer"
	repo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repos"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-outbox/pkg/types"
	"github.com/sirupsen/logrus"
)

type Server struct {
	consumer *consumer.KafkaConsumer[repo.Event]
	msgCH    chan *pkgtypes.Message[repo.Event]
}

func NewServer(eventRepo *repo.EventRepo) *Server {
	return &Server{
		msgCH: make(chan *pkgtypes.Message[repo.Event], 64),
	}
}

func (s *Server) addConsumer() *Server {
	c := consumer.NewKafkaConsumer(s.msgCH)
	s.consumer = c
	return s
}

func (s *Server) handleMsg(msg *pkgtypes.Message[repo.Event]) {
	<-s.consumer.ReadyCH
	logrus.WithFields(logrus.Fields{
		"EventID": msg.Data.EventId,
		"Offset":  msg.Metadata.Offset,
		"PRTN":    msg.Metadata.Partition,
	}).Info("MSG:RECEIVED")
	// r := time.Duration(rand.IntN(5))
	// time.Sleep(r * time.Second)
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	// defer cancel()
	// _, err := s.saveToDB(ctx, msg)
	// if err != nil {
	// 	// fmt.Printf("ERR on DB SAVE = %v\n", err)
	// 	return
	// }
	// // fmt.Printf("INSERT SUCCESS for OFFSET = %d, PRTN = %d, EventID = %s\n", msg.Metadata.Offset, msg.Metadata.Partition, id)
}

func main() {
	db, err := dbpostgres.NewDBConn(&dbpostgres.DBPostgresOptions{
		DBname: "audit",
	})
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()

	er := repo.NewEventRepo(db)
	s := NewServer(er).addConsumer()

	go func() {
		for msg := range s.msgCH {
			go s.handleMsg(msg)
		}
	}()

	s.consumer.RunConsumer()
}

// func (s *Server) saveToDB(ctx context.Context, msg *shared.Message) (string, error) {
// 	return repo.TxClosure(ctx, s.eventRepo, func(ctx context.Context, tx *sqlx.Tx) (string, error) {
// 		// fmt.Printf("starting DB operation for OFFSET = %d, PRTN = %d, EventID = %s\n", msg.Metadata.Offset, msg.Metadata.Partition, msg.Event.EventId)
// 		id, err := s.eventRepo.Insert(ctx, tx, msg.Event)
// 		if err != nil {
// 			exists := repo.IsDuplicateKeyErr(err)
// 			if exists {
// 				eMsg := fmt.Sprintf("already exists OFFSET = %d, PRTN = %d, EventID = %s\n", msg.Metadata.Offset, msg.Metadata.Partition, msg.Event.EventId)
// 				s.consumer.UpdateState(msg.Metadata, shared.MsgState_Success)
// 				return "", errors.New(eMsg)
// 			}
// 			s.consumer.UpdateState(msg.Metadata, shared.MsgState_Error)
// 			return "", err
// 		}
// 		s.consumer.UpdateState(msg.Metadata, shared.MsgState_Success)
// 		return id, nil
// 	})
// }
