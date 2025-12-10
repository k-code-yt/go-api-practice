package outbox

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/producer"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/event"
)

type OutboxService struct {
	eventRepo *event.EventRepo
	producer  *producer.KafkaProducer
	consumer  *consumer.KafkaConsumer
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

func (s *OutboxService) addConsumer() *OutboxService {
	c := consumer.NewKafkaConsumer(s.msgCH)
	s.consumer = c
	return s
}

func (s *OutboxService) outboxLoop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
		}
	}
}

func (s *OutboxService) handlePending() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	s.eventRepo.GetAllPending(ctx)
}
