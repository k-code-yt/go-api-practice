package pkgkafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Message[T any] struct {
	Metadata *kafka.TopicPartition
	Data     T
	cfg      *KafkaConfig
}

func NewMessage[T any](metadata *kafka.TopicPartition, payload T) (*Message[T], error) {
	return &Message[T]{
		Metadata: metadata,
		Data:     payload,
	}, nil
}
