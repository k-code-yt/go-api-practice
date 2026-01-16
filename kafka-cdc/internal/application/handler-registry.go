package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure"
	libtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/lib/types"
	kafkainternal "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka_internal"
)

type Handler func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error

type Registry struct {
	handlers map[libtypes.EventType]Handler
	mu       *sync.RWMutex
	consumer *kafkainternal.KafkaConsumer
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[libtypes.EventType]Handler),
		mu:       new(sync.RWMutex),
	}
}

func (r *Registry) InitHandlers() struct{} {
	go c.checkReadyToAccept()
	go c.consumeLoop()
	return <-c.exitCH
}

func (r *Registry) AddHandler(h Handler, event libtypes.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[event] = h
}

func (r *Registry) GetHandlerFromMsg(msg *kafka.Message) (Handler, error) {
	topic := *msg.TopicPartition.Topic

	parsed := &infrastructure.PartialDebeziumMessage{}
	err := json.Unmarshal(msg.Value, parsed)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse CDC msg %v", err)

	}

	event, err := infrastructure.GetEventType(topic, parsed.Payload.Op)
	if err != nil {
		return nil, err
	}

	parsed = nil
	return r.getHandler(libtypes.EventType(event))
}

func (r *Registry) getHandler(event libtypes.EventType) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[event]
	if !ok {
		return nil, fmt.Errorf("handler is missing for %s event type", event)
	}
	return h, nil
}

func (r *Registry) getHandler(event libtypes.EventType) (Handler, error) {
	defer func() {
		r.consumer.Close()
		close(r.consumer.exitCH)
	}()
	firstMsg := true

	for {
		msg, err := r.consumer.ReadMessage(time.Second)
		if err != nil && err.(kafka.Error).IsTimeout() {
			continue
		}
		if err != nil && !err.(kafka.Error).IsTimeout() {
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}
		if msg == nil {
			continue
		}

		if firstMsg {
			close(r.consumer.ReadyCH)
		}

		firstMsg = false

		c.appendMsgState(&msg.TopicPartition)

		handler, err := c.HandlerRegistry.GetHandlerFromMsg(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = handler(context.Background(), msg.Value, &msg.TopicPartition)
		if err != nil {
			fmt.Println(err)
			c.UpdateState(&msg.TopicPartition, MsgState_Error)
			continue
		}
	}

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

func convertIntTimeToUnix(microSeconds int64) time.Time {
	seconds := microSeconds / 1_000_000
	nanos := (microSeconds % 1_000_000) * 1000
	return time.Unix(int64(seconds), int64(nanos))
}
