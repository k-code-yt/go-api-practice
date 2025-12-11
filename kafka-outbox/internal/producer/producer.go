package producer

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/shared"
)

type KafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer() *KafkaProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}

	go func() {
		defer p.Close()
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to PRTN = %d OFF = %d\n", ev.TopicPartition.Partition, ev.TopicPartition.Offset)
				}
			}
		}
	}()

	return &KafkaProducer{
		producer: p,
	}
}

func (p *KafkaProducer) Produce(msg []byte) error {
	cfg := shared.NewKafkaConfig()
	topic := cfg.DefaultTopic
	return p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          msg,
	}, nil)
}
