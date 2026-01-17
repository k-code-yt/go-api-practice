package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type Handler func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition, eventType pkgtypes.EventType) error

type MsgRouter struct {
	handlers map[pkgtypes.EventType]Handler
	mu       *sync.RWMutex
	consumer *pkgkafka.KafkaConsumer
}

func NewMsgRouter(consumer *pkgkafka.KafkaConsumer) *MsgRouter {
	r := &MsgRouter{
		handlers: make(map[pkgtypes.EventType]Handler),
		mu:       new(sync.RWMutex),
		consumer: consumer,
	}

	go r.consumer.RunConsumer()
	go r.consumeLoop()
	return r
}

func (r *MsgRouter) consumeLoop() {
	for msg := range r.consumer.MsgCH {
		go r.handlerMsg(msg)
	}
}

func (r *MsgRouter) AddHandler(h Handler, event pkgtypes.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[event] = h
}

func (r *MsgRouter) handlerMsg(msg *kafka.Message) error {
	topic := *msg.TopicPartition.Topic

	parsed := &debezium.PartialDebeziumMessage{}
	err := json.Unmarshal(msg.Value, parsed)
	if err != nil {
		return fmt.Errorf("Cannot parse CDC msg %v", err)

	}

	event, err := r.getEventType(topic, parsed.Payload.Op)
	if err != nil {
		return err
	}

	parsed = nil
	handler, err := r.getHandler(pkgtypes.EventType(event))
	if err != nil {
		return err
	}
	return handler(context.Background(), msg.Value, &msg.TopicPartition, event)
}

func (r *MsgRouter) getEventType(topic string, op string) (pkgtypes.EventType, error) {
	eventRaw := strings.Split(topic, ".")
	if topic == "" || len(eventRaw) < 3 {
		return "", fmt.Errorf("Invalid topic & event type combination")
	}

	event := eventRaw[2]
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

	return pkgtypes.EventType(event), nil
}

func (r *MsgRouter) getHandler(event pkgtypes.EventType) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[event]
	if !ok {
		return nil, fmt.Errorf("handler is missing for %s event type", event)
	}
	return h, nil
}
