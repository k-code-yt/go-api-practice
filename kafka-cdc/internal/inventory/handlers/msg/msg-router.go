package handlersmsg

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	inframsg "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/infra/msg"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgerrors "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/errors"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	pkgproto "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types/proto"
)

type MsgRouter struct {
	consumer       *pkgkafka.KafkaConsumer
	mu             *sync.RWMutex
	paymentHandler *PaymentHandler
}

func NewMsgRouter(consumer *pkgkafka.KafkaConsumer, ph *PaymentHandler) *MsgRouter {
	r := &MsgRouter{
		consumer:       consumer,
		mu:             new(sync.RWMutex),
		paymentHandler: ph,
	}

	go r.consumer.RunConsumer()
	go r.consumeLoop()
	return r
}

func (r *MsgRouter) HandlePaymentCreated(ctx context.Context, paymentEvent *domain.PaymentCreatedEvent, eventType pkgtypes.EventType) error {
	return r.paymentHandler.handlePaymentCreated(ctx, paymentEvent, eventType)
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

	msgEncoderType := r.consumer.MsgEncoder.GetType()
	switch eventType {
	case pkgconstants.EventType_PaymentCreated:
		msgReq, err := r.paymentHandler.getPaymentCreatedEvent(context.Background(), envelope, eventType, msgEncoderType, &msg.TopicPartition)
		if err != nil {
			fmt.Println(err)
			return err
		}
		return r.HandlePaymentCreated(msgReq.Ctx, msgReq.Payload.After, eventType)
	}
	return nil
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
		target := &pkgproto.CDCPaymentEnvelope{}
		err := r.consumer.MsgEncoder.Decoder(&msg.TopicPartition, msg.Value, target)
		if err != nil {
			return nil, "", err
		}
		return target, target.Op, nil
	}
	return nil, "", fmt.Errorf("Unknown topic, missing handler for it %s\n", *msg.TopicPartition.Topic)

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
