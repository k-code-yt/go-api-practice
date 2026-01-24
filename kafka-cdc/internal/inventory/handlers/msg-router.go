package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	inframsg "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/infra/msg"
	pkgerrors "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/errors"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type Handler[T any] func(ctx context.Context, payload T, metadata *kafka.TopicPartition, eventType pkgtypes.EventType, cfg *pkgkafka.KafkaConfig) error

type MsgRouter struct {
	consumer *pkgkafka.KafkaConsumer
}

func NewMsgRouter(consumer *pkgkafka.KafkaConsumer) *MsgRouter {
	r := &MsgRouter{
		consumer: consumer,
	}

	go r.consumer.RunConsumer()
	go r.consumeLoop()
	return r
}

func (r *MsgRouter) consumeLoop() {
	for msg := range r.consumer.MsgCH {
		go func() {
			err := r.handlerMsg(msg)
			if err != nil && !pkgerrors.IsDuplicateKeyError(err) {
				r.consumer.UpdateState(&msg.TopicPartition, pkgkafka.MsgState_Error)
				return
			}
			r.consumer.UpdateState(&msg.TopicPartition, pkgkafka.MsgState_Success)
		}()
	}
}

func (r *MsgRouter) decodeMsg(msg *kafka.Message) (any, string, error) {
	topic := *msg.TopicPartition.Topic

	switch topic {
	case inframsg.DebPaymentTopic:
		return r.decodePayment(msg)
	}
	return nil, "", fmt.Errorf("Unknown topic, missing handler for it %s\n", topic)
}

func (r *MsgRouter) decodePayment(msg *kafka.Message) (any, string, error) {
	msgEncoderType := r.consumer.MsgEncoder.GetType()

	if msgEncoderType == pkgkafka.KafkaEncoder_PROTO {
		target := &pkgtypes.CDCPaymentEnvelope{}
		err := r.consumer.MsgEncoder.Decoder(&msg.TopicPartition, msg.Value, target)
		if err != nil {
			return nil, "", err
		}
		return target, target.Op, nil
	}
	return nil, "", fmt.Errorf("Unknown topic, missing handler for it %s\n", *msg.TopicPartition.Topic)

}

func (r *MsgRouter) handlerMsg(msg *kafka.Message) error {
	topic := *msg.TopicPartition.Topic

	envelope, op, err := r.decodeMsg(msg)
	if err != nil {
		return err
	}

	eventType, err := r.getEventType(topic, op)
	if err != nil {
		return err
	}

	fmt.Println(eventType)
	fmt.Println(envelope)

	// msgEncoderType := r.consumer.MsgEncoder.GetType()

	// switch msgEncoderType {
	// case pkgkafka.KafkaEncoder_PROTO:

	// }
	return nil
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
