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

type Payload[T any] struct {
	Before      *T                   `avro:"before"`
	After       *T                   `avro:"after"`
	Source      DebeziumSource       `avro:"source"`
	Transaction *DebeziumTransaction `avro:"transaction"`
	Op          string               `avro:"op"`
	Timestamp   *int64               `avro:"ts_ms"`
	EventType   pkgtypes.EventType
}

type DebeziumSource struct {
	Version   string  `avro:"version"`
	Connector string  `avro:"connector"`
	Name      string  `avro:"name"`
	TsMs      int64   `avro:"ts_ms"`
	Snapshot  *string `avro:"snapshot"`
	Db        string  `avro:"db"`
	Sequence  *string `avro:"sequence"`
	TsUs      *int64  `avro:"ts_us"`
	TsNs      *int64  `avro:"ts_ns"`
	Schema    string  `avro:"schema"`
	Table     string  `avro:"table"`
	TxId      *int64  `avro:"txId"`
	Lsn       *int64  `avro:"lsn"`
	Xmin      *int64  `avro:"xmin"`
}

type DebeziumTransaction struct {
	Id                  string `json:"id" avro:"id"`
	TotalOrder          int64  `json:"total_order" avro:"total_order"`
	DataCollectionOrder int64  `json:"data_collection_order" avro:"data_collection_order"`
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
