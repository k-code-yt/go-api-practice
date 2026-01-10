package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
)

type Handler func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error

type Registry struct {
	handlers map[pkgconstants.EventType]Handler
	mu       *sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[pkgconstants.EventType]Handler),
		mu:       new(sync.RWMutex),
	}
}

func (r *Registry) AddHandler(h Handler, event pkgconstants.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[event] = h
}

func (r *Registry) GetHandlerFromMsg(msg *kafka.Message) (Handler, error) {
	topic := *msg.TopicPartition.Topic

	parsed := &debezium.PartialDebeziumMessage{}
	err := json.Unmarshal(msg.Value, parsed)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse CDC msg %v", err)

	}

	event, err := debezium.GetEventType(topic, parsed.Payload.Op)
	if err != nil {
		return nil, err
	}

	parsed = nil
	return r.getHandler(pkgconstants.EventType(event))
}

func (r *Registry) getHandler(event pkgconstants.EventType) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[event]
	if !ok {
		return nil, fmt.Errorf("handler is missing for %s event type", event)
	}
	return h, nil
}

func decodeDebeziumDecimal(encoded string, scale int) (float64, error) {
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
