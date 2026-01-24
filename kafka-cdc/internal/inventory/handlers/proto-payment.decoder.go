package handlers

import (
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/protobuf"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

func ProtoParseDebeziumPayload(deser *protobuf.Deserializer, topic string, data []byte) (*pkgtypes.CDCPaymentEnvelope, error) {
	var payload pkgtypes.CDCPaymentEnvelope

	err := deser.DeserializeInto(topic, data, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
