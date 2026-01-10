package debezium

import (
	"context"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
)

type DebeziumMessage[T any] struct {
	Ctx      context.Context
	Payload  Payload[T]
	Metadata *kafka.TopicPartition
	// Schema  json.RawMessage
}

func (msg *DebeziumMessage[T]) AddMetadata(m *kafka.TopicPartition) {
	msg.Metadata = m
}

type Payload[T any] struct {
	Before    T
	After     T
	Source    Source
	Op        string `json:"op"`
	Timestamp int    `json:"ts_ms"`
	EventType pkgconstants.EventType
}

type Source struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	Timestamp int    `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	Sequence  string `json:"sequence"`
	Table     string `json:"table"`
	TxId      int    `json:"txId"`
	Lsn       int    `json:"lsn"`
}

type PartialPayload struct {
	Op string `json:"op"`
}

type PartialDebeziumMessage struct {
	Payload PartialPayload `json:"payload"`
}

func (p *Payload[T]) AddEventType(topic string) error {
	event, err := GetEventType(topic, p.Op)
	if err != nil {
		return err
	}
	p.EventType = event
	return nil
}

func GetEventType(topic string, op string) (pkgconstants.EventType, error) {
	r := strings.Split(topic, ".")
	if topic == "" || len(r) < 3 {
		return "", fmt.Errorf("Invalid topic & event type combination")
	}

	event := r[2]
	switch op {
	case "c", "r":
		event = fmt.Sprintf("%s_%s", event, "created")
	case "u":
		event = fmt.Sprintf("%s_%s", event, "updated")
	case "d":
		event = fmt.Sprintf("%s_%s", event, "deleted")
	case "t":
		event = fmt.Sprintf("%s_%s", event, "truncated")
	}

	return pkgconstants.EventType(event), nil
}
