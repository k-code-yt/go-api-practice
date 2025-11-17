package shared

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

type Message struct {
	Metadata *kafka.TopicPartition
	Data     string
}

func NewMessage(metadata *kafka.TopicPartition, data []byte) *Message {
	return &Message{
		Metadata: metadata,
		Data:     string(data),
	}
}
