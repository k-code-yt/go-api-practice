package handlersmsg

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	inventory "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type PaymentHandler struct {
	svc         *application.InventoryService
	protoMapper *ProtoMapper
	avroMapper  *AvroMapper
}

func NewPaymentHandler(svc *application.InventoryService) *PaymentHandler {
	h := &PaymentHandler{
		svc:         svc,
		protoMapper: NewProtoMapper(),
		avroMapper:  NewAvroMapper(),
	}
	return h
}

func (h *PaymentHandler) getPaymentCreatedEvent(ctx context.Context, envelope any, eventType pkgtypes.EventType, encoderType pkgkafka.KafkaEncoder, metadata *kafka.TopicPartition) (*debezium.DebeziumMessage[domain.PaymentCreatedEvent], error) {
	switch encoderType {
	case pkgkafka.KafkaEncoder_PROTO:
		return h.protoMapper.getPaymentCreatedEvent(ctx, envelope, eventType, metadata)
	}
	return nil, fmt.Errorf("unknown eventType %s, encoderType %s combo", eventType, encoderType)
}

func (h *PaymentHandler) handlePaymentCreated(ctx context.Context, paymentEvent *domain.PaymentCreatedEvent, eventType pkgtypes.EventType) error {
	inv, inbox, err := inventory.PaymentToInventory(paymentEvent, eventType)
	if err != nil {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
		return err
	}

	inboxID, err := h.svc.Save(ctx, inbox, inv)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inbox.AggregateId,
			},
		).Error("INSERT:ERROR")
		return err
	}
	logrus.WithFields(
		logrus.Fields{
			"inboxID":     inboxID,
			"aggregateID": inbox.AggregateId,
		},
	).Info("INSERT:SUCCESS")
	return nil
}
