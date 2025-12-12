package pkgtypes

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Message[T any] struct {
	Metadata *kafka.TopicPartition
	Data     T
}

// TODO -> make generic
func NewMessage[T any](metadata *kafka.TopicPartition, data []byte) (*Message[T], error) {
	var payload T
	err := json.Unmarshal(data, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &Message[T]{
		Metadata: metadata,
		Data:     payload,
	}, nil
}

func MustNewMessage[T any](metadata *kafka.TopicPartition, data []byte) *Message[T] {
	msg, err := NewMessage[T](metadata, data)
	if err != nil {
		panic(err)
	}
	return msg
}
