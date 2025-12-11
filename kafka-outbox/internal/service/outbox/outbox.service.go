package outbox

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/producer"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/event"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/repo-shared"
)

type OutboxService struct {
	eventRepo *event.EventRepo
	producer  *producer.KafkaProducer
	// consumer  *consumer.KafkaConsumer
}

func NewOutbox(er *event.EventRepo) *OutboxService {
	s := &OutboxService{
		eventRepo: er,
	}

	s.addProducer()
	go s.outboxLoop()
	return s
}

func (s *OutboxService) addProducer() *OutboxService {
	shouldProduce := os.Getenv("SHOULD_PRODUCE") == "true"
	if !shouldProduce {
		shouldProduce = *flag.Bool("SHOULD_PRODUCE", false, "Enable message production")
		flag.Parse()
	}
	fmt.Printf("SHOULD_PRODUCE = %t\n", shouldProduce)

	if shouldProduce {
		s.producer = producer.NewKafkaProducer()
	}
	return s
}

func (s *OutboxService) outboxLoop() {
	// TODO -> adjust dur > 30sec
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for range ticker.C {
		s.handlePending()
	}

}
func (s *OutboxService) handlePending() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	repo := s.eventRepo.GetRepo()
	_, err := reposhared.TxClosure(ctx, repo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		events, err := s.eventRepo.GetAllPending(ctx, tx)
		if err != nil {
			return 0, err
		}

		toUpdateIds := []string{}
		for _, e := range events {
			// TODO -> add batch processing
			// TODO -> add binary protocol
			b, err := json.Marshal(e)
			if err != nil {
				fmt.Printf("error marshalling event %v\n", err)
				continue
			}
			err = s.producer.Produce(b)
			if err != nil {
				fmt.Printf("error producing event %v\n", err)
				continue
			}
			toUpdateIds = append(toUpdateIds, e.EventId)
		}

		rows, err := s.eventRepo.UpdateStatusByIds(ctx, tx, toUpdateIds, event.EventStatus_Produced)
		if err != nil {
			return 0, err
		}
		if rows != len(toUpdateIds) {
			errMsg := "updated row count didn't match"
			fmt.Println(errMsg)
			return 0, fmt.Errorf("%v", errMsg)
		}
		return rows, nil
	})

	if err != nil {
		fmt.Printf("err outbox service %v\n", err)
	}
}

// func (s *OutboxService) addConsumer() *OutboxService {
// 	c := consumer.NewKafkaConsumer(s.msgCH)
// 	s.consumer = c
// 	return s
// }

// func (s *OutboxService) handlePending() {
// 	ticker := time.NewTicker(time.Second * 5)

// 	for {
// 		select {
// 		case <-ticker.C:
// 		}
// 	}
// }
