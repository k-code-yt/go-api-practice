package handlers

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
)

func AvroParseDebeziumPayload(deserializer *avro.SpecificDeserializer, topic string, data []byte) (*debezium.Payload[CDCPayment], error) {
	var payload debezium.Payload[CDCPayment]

	err := deserializer.DeserializeInto(topic, data, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

func AvroGnericParseDebeziumPayload(deserializer *avro.GenericDeserializer, topic string, data []byte) (*debezium.Payload[CDCPayment], error) {
	var payload debezium.Payload[CDCPayment]
	// schema, err := deserializer.GetSchema(topic, data)
	// if err != nil {
	// 	return nil, err
	// }

	// var jSchema map[string]any
	// err = json.Unmarshal([]byte(schema.Schema), &jSchema)
	// if err != nil {
	// 	return nil, err
	// }
	DebugSchema("http://localhost:8081", topic)
	err := deserializer.DeserializeInto(topic, data, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

func DebugSchema(schemaRegistryURL, topic string) {
	client, _ := schemaregistry.NewClient(schemaregistry.NewConfig(schemaRegistryURL))

	// Get the latest schema for the topic
	subject := topic + "-value"
	schemaMetadata, err := client.GetLatestSchemaMetadata(subject)
	if err != nil {
		panic(err)
	}

	fmt.Println("Schema from registry:")
	fmt.Println(schemaMetadata.Schema)
}
