package pkgkafka

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
)

type KafkaEncoder string

const (
	KafkaEncoder_JSON  KafkaEncoder = "json"
	KafkaEncoder_AVRO  KafkaEncoder = "avro"
	KafkaEncoder_PROTO KafkaEncoder = "proto"
)

type Decoder func(metadata *kafka.TopicPartition, data []byte, target any) error

type MsgEncoder interface {
	Decoder(metadata *kafka.TopicPartition, data []byte, target any) error
	GetType() KafkaEncoder
}

type SchemaRegistryConfig struct {
	URL         string
	EncoderType KafkaEncoder
}

var DefaultSchemaRegistryConfig = SchemaRegistryConfig{
	URL:         "http://localhost:8081",
	EncoderType: KafkaEncoder_PROTO,
}

func NewMsgEncoder(srConfig *SchemaRegistryConfig) (MsgEncoder, error) {
	var encoder MsgEncoder
	var err error
	if srConfig == nil {
		srConfig = &DefaultSchemaRegistryConfig
	}

	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig(srConfig.URL))
	if err != nil {
		fmt.Printf("Unable to create schema registry %v\n", srClient)
	}

	switch srConfig.EncoderType {
	case KafkaEncoder_AVRO:
		encoder, err = NewAvroEncoder(srClient, srConfig.URL)
	case KafkaEncoder_PROTO:
		encoder, err = NewProtoEncoder(srClient, srConfig.URL)
	default:
		encoder = NewJsonEncoder()
	}
	return encoder, err
}

type JsonEncoder struct {
	msgEncoderType KafkaEncoder
}

func NewJsonEncoder() *JsonEncoder {
	return &JsonEncoder{
		msgEncoderType: KafkaEncoder_JSON,
	}
}

func (e *JsonEncoder) Decoder(metadata *kafka.TopicPartition, data []byte, payload any) error {
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return nil
}

func (e *JsonEncoder) GetType() KafkaEncoder {
	return e.msgEncoderType
}
