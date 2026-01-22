package pkgkafka

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/schemaregistry/serde/avro"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Decoder[T any] func(metadata *kafka.TopicPartition, data []byte) (T, error)

type Message[T any] struct {
	Metadata *kafka.TopicPartition
	Data     T
	cfg      *KafkaConfig
	decoder  Decoder[T]
}

func NewMessage[T any](decoder Decoder[T], metadata *kafka.TopicPartition, data []byte) (*Message[T], error) {
	payload, err := decoder(metadata, data)
	if err != nil {
		return nil, err
	}

	return &Message[T]{
		Metadata: metadata,
		Data:     payload,
	}, nil
}

func jsonDecoder[T any](metadata *kafka.TopicPartition, data []byte) (T, error) {
	var payload T
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return payload, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return payload, nil
}

func CreateAvroDecoder[T any](deser *avro.SpecificDeserializer) Decoder[T] {
	return func(metadata *kafka.TopicPartition, data []byte) (T, error) {
		var payload T
		err := deser.DeserializeInto(*metadata.Topic, data, &payload)
		if err != nil {
			return payload, fmt.Errorf("Deserialization error: %v\n", err)
		}

		return payload, nil
	}
}
