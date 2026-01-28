package pkgkafka

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
	goavro "github.com/linkedin/goavro/v2"
)

type AvroSchemaCache struct {
	mu     *sync.RWMutex
	codecs map[int]*goavro.Codec
}

func NewAvroSchemaCache() *AvroSchemaCache {
	return &AvroSchemaCache{
		mu:     new(sync.RWMutex),
		codecs: make(map[int]*goavro.Codec),
	}
}

func (sc *AvroSchemaCache) GetCodec(srClient schemaregistry.Client, topic string, schemaID int) (*goavro.Codec, error) {
	sc.mu.RLock()
	codec, ok := sc.codecs[schemaID]
	sc.mu.RUnlock()

	if ok {
		return codec, nil
	}

	subject := topic + "-value"
	schemaInfo, err := srClient.GetBySubjectAndID(subject, schemaID)
	if err != nil {
		return nil, err
	}

	codec, err = goavro.NewCodec(schemaInfo.Schema)
	if err != nil {
		return nil, err
	}

	sc.mu.Lock()
	if sc.codecs == nil {
		sc.codecs = make(map[int]*goavro.Codec)
	}
	sc.codecs[schemaID] = codec
	sc.mu.Unlock()

	return codec, nil
}

type AvroEncoder struct {
	msgEncoderType KafkaEncoder

	schemaCache          *AvroSchemaCache
	schemaRegistryClient schemaregistry.Client
	schemaRegistryURL    string
	specificDeser        *avro.SpecificDeserializer
	genericDeser         *avro.GenericDeserializer
}

func NewAvroEncoder(schemaRegistryClient schemaregistry.Client, schemaRegistryURL string) (*AvroEncoder, error) {
	encoder := AvroEncoder{
		msgEncoderType: KafkaEncoder_AVRO,
	}

	srClient, err := schemaregistry.NewClient(schemaregistry.NewConfig(schemaRegistryURL))
	if err != nil {
		return nil, err
	}

	specificDeser, err := avro.NewSpecificDeserializer(srClient, serde.ValueSerde, avro.NewDeserializerConfig())
	if err != nil {
		return nil, err
	}
	encoder.specificDeser = specificDeser

	genericDeser, err := avro.NewGenericDeserializer(srClient, serde.ValueSerde, avro.NewDeserializerConfig())
	if err != nil {
		return nil, err
	}
	encoder.genericDeser = genericDeser

	encoder.schemaRegistryClient = srClient
	encoder.schemaCache = NewAvroSchemaCache()

	return &encoder, nil
}

func (e *AvroEncoder) Decoder(metadata *kafka.TopicPartition, data []byte, target any) error {
	err := e.genericDeser.DeserializeInto(*metadata.Topic, data, target)
	if err != nil {
		return fmt.Errorf("Deserialization error: %v\n", err)
	}
	return nil
}

func (e *AvroEncoder) GetType() KafkaEncoder {
	return e.msgEncoderType
}
