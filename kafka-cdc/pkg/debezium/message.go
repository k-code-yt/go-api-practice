package debezium

import (
	"context"
	"encoding/base64"
	"math/big"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type DebeziumMessage[T any] struct {
	Ctx      context.Context
	Payload  Payload[T]
	Metadata *kafka.TopicPartition
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
	EventType pkgtypes.EventType
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

func (p *Payload[T]) AddEventType(event pkgtypes.EventType) error {
	p.EventType = event
	return nil
}

func DecodeDebeziumDecimal(encoded string, scale int) (float64, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, err
	}

	bigInt := new(big.Int).SetBytes(decoded)

	bigFloat := new(big.Float).SetInt(bigInt)

	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	result := new(big.Float).Quo(bigFloat, divisor)

	val, _ := result.Float64()
	return val, nil
}

func ConvertIntTimeToUnix(microSeconds int64) time.Time {
	seconds := microSeconds / 1_000_000
	nanos := (microSeconds % 1_000_000) * 1000
	return time.Unix(int64(seconds), int64(nanos))
}
