package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka/internal/consumer"
	"github.com/k-code-yt/go-api-practice/kafka/internal/producer"
	"github.com/k-code-yt/go-api-practice/kafka/internal/repo"
	"github.com/k-code-yt/go-api-practice/kafka/internal/shared"
)

type Server struct {
	addr      string
	producer  *producer.KafkaProducer
	consumer  *consumer.KafkaConsumer
	eventRepo *repo.EventRepo
}

func NewServer(addr string, eventRepo *repo.EventRepo) *Server {
	return &Server{
		addr:      addr,
		producer:  producer.NewKafkaProducer(),
		consumer:  consumer.NewKafkaConsumer(),
		eventRepo: eventRepo,
	}
}

func (s *Server) handleMsg(msg *shared.Message) {
	r := time.Duration(rand.IntN(5))
	time.Sleep(r * time.Second)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	s.saveToDB(ctx, msg)
}

func (s *Server) saveToDB(ctx context.Context, msg *shared.Message) {
	repo.TxClosure(ctx, s.eventRepo, func(ctx context.Context, tx *sqlx.Tx) (string, error) {
		fmt.Printf("starting DB operation for OFFSET = %d, EventID = %s\n", msg.Metadata.Offset, msg.Event.EventId)
		// TODO -> how to handle insert error
		defer s.consumer.MarkAsComplete(msg.Metadata)
		event := s.eventRepo.Get(ctx, tx, msg.Event.EventId)
		if event != nil {
			eMsg := fmt.Sprintf("offset = %d, eventID %s already existing -> skipping\n", msg.Metadata.Offset, msg.Event.EventId)
			return "", errors.New(eMsg)
		}

		id, err := s.eventRepo.Insert(ctx, tx, msg.Event)
		if err != nil {
			return "", err
		}
		fmt.Printf("INSERT SUCCESS, EventID = %s, Offset = %d\n", id, msg.Metadata.Offset)
		return id, nil
	})

}

func (s *Server) produceMsgs() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		event := repo.NewEvent()
		b, err := json.Marshal(event)
		if err != nil {
			fmt.Printf("err marshaling event = %v\n", err)
			continue
		}
		s.producer.Produce(string(b))
	}
}

func main() {
	db, err := repo.NewDBConn()
	if err != nil {
		panic(fmt.Sprintf("unable to conn to db, err = %v\n", err))
	}
	defer db.Close()
	er := repo.NewEventRepo(db)
	s := NewServer(":7576", er)
	go s.produceMsgs()
	for msg := range s.consumer.MsgCH {
		go s.handleMsg(msg)
	}
}
