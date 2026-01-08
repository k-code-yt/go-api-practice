package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type PaymentCreatedHandler struct {
	Handler Handler
	MsgCH   chan repo.PaymentCreatedEvent
}

func NewPaymentCreatedHandler() *PaymentCreatedHandler {
	h := &PaymentCreatedHandler{
		MsgCH: make(chan repo.PaymentCreatedEvent, 64),
	}
	handlerFunc := h.CreateHandlerFunc()
	h.Handler = handlerFunc
	return h

}

func (h *PaymentCreatedHandler) CreateHandlerFunc() Handler {
	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error {
		msgRequest, err := pkgtypes.NewMessage[debezium.DebeziumMessage[repo.PaymentCreatedEvent]](metadata, msg)
		if err != nil {
			return err
		}

		err = msgRequest.Data.Payload.AddEventType(*metadata.Topic)
		if err != nil {
			return err
		}
		fmt.Println(msgRequest)

		select {
		case h.MsgCH <- msgRequest.Data.Payload.After:

		case <-time.After(5 * time.Second):
			logrus.Errorf("MsgCH blocked for 10s, dropping message offset=%d partition=%d",
				metadata.Offset, metadata.Partition)
			// c.UpdateState(&msg.TopicPartition, MsgState_Error)
		}

		return nil
	}

}
