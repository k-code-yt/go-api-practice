package pkgkafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/protobuf"
)

type ProtoEncoder struct {
	msgEncoderType KafkaEncoder

	schemaRegistryClient schemaregistry.Client
	schemaRegistryURL    string
	deser                *protobuf.Deserializer
}

func NewProtoEncoder(schemaRegistryClient schemaregistry.Client, schemaRegistryURL string) (*ProtoEncoder, error) {
	encoder := ProtoEncoder{
		msgEncoderType: KafkaEncoder_PROTO,
	}

	deser, err := protobuf.NewDeserializer(schemaRegistryClient, serde.ValueSerde, protobuf.NewDeserializerConfig())
	if err != nil {
		fmt.Printf("failed to create deserializer: %v", err)
		return nil, err
	}
	encoder.deser = deser

	return &encoder, nil
}

func (e *ProtoEncoder) Decoder(metadata *kafka.TopicPartition, data []byte, target any) error {
	err := e.deser.DeserializeInto(*metadata.Topic, data, target)
	if err != nil {
		return fmt.Errorf("Deserialization error: %v\n", err)
	}
	return nil
}

func (e *ProtoEncoder) GetType() KafkaEncoder {
	return e.msgEncoderType
}
