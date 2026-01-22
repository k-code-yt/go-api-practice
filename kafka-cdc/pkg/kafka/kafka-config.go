package pkgkafka

import (
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/schemaregistry/serde/avro"
)

type KafkaEncoder string

const (
	KafkaEncoder_JSON KafkaEncoder = "json"
	KafkaEncoder_AVRO KafkaEncoder = "avro"
)

type KafkaConfig struct {
	DefaultTopics            []string
	TopicsCount              int
	Host                     string
	ConsumerGroup            string
	ParititionAssignStrategy string
	NumPartitions            int
	MsgEncoderType           KafkaEncoder
	AvroDeserializer         *avro.SpecificDeserializer
}

func GetDecoder[T any](cfg *KafkaConfig) Decoder[T] {
	if cfg.MsgEncoderType == KafkaEncoder_AVRO {
		return CreateAvroDecoder[T](cfg.AvroDeserializer)
	}
	return jsonDecoder
}

func NewKafkaConfig() *KafkaConfig {
	// TODO -> move to func arg
	msgEncoder := KafkaEncoder_AVRO

	strategy := "cooperative-sticky"
	count := 1
	topics := []string{}
	for id := range count {
		t := fmt.Sprintf("%s_topic_%d", strategy, id)
		topics = append(topics, t)
	}

	cfg := &KafkaConfig{
		ParititionAssignStrategy: strategy,
		DefaultTopics:            topics,
		Host:                     "localhost",
		ConsumerGroup:            "local_cg",
		NumPartitions:            4,
		MsgEncoderType:           msgEncoder,
	}

	if msgEncoder == KafkaEncoder_AVRO {
		srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig(
			"http://localhost:8081",
		))
		if err != nil {
			log.Fatal(err)
		}

		deser, err := avro.NewSpecificDeserializer(srClient, serde.ValueSerde, avro.NewDeserializerConfig())
		if err != nil {
			log.Fatal(err)
		}
		cfg.AvroDeserializer = deser
	}

	return cfg
}
