package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	inventory "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type CDCPayment struct {
	ID          int    `json:"id"`
	OrderNumber string `json:"order_number"`
	Amount      string `json:"amount"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func (cdc *CDCPayment) toEvent() (*domain.PaymentCreatedEvent, error) {
	amount, err := strconv.ParseFloat(cdc.Amount, 2)
	if err != nil {
		amount, err = debezium.DecodeDebeziumDecimal(cdc.Amount, 2)
		if err != nil {
			return nil, err
		}
	}
	return &domain.PaymentCreatedEvent{
		ID:          cdc.ID,
		OrderNumber: cdc.OrderNumber,
		Amount:      amount,
		Status:      cdc.Status,
		CreatedAt:   debezium.ConvertIntTimeToUnix(cdc.CreatedAt),
		UpdatedAt:   debezium.ConvertIntTimeToUnix(cdc.UpdatedAt),
	}, nil
}

type PaymentHandler struct {
	Handler Handler
	svc     *application.InventoryService
}

func NewPaymentCreatedHandler(svc *application.InventoryService) *PaymentHandler {
	h := &PaymentHandler{
		svc: svc,
	}
	handlerFunc := h.CreateHandlerFunc()
	h.Handler = handlerFunc
	return h

}

func (h *PaymentHandler) CreateHandlerFunc() Handler {
	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition, eventType pkgtypes.EventType) error {
		parsed, err := pkgkafka.NewMessage[debezium.DebeziumMessage[CDCPayment]](metadata, msg)
		if err != nil {
			return err
		}
		msgRequest := debezium.DebeziumMessage[domain.PaymentCreatedEvent]{
			Payload: debezium.Payload[domain.PaymentCreatedEvent]{
				Source:    parsed.Data.Payload.Source,
				Op:        parsed.Data.Payload.Op,
				Timestamp: parsed.Data.Payload.Timestamp,
				EventType: parsed.Data.Payload.EventType,
			},
			Metadata: metadata,
			Ctx:      ctx,
		}
		after, err := parsed.Data.Payload.After.toEvent()
		msgRequest.Payload.After = *after
		if err != nil {
			return err
		}
		if parsed.Data.Payload.Before.ID != 0 {
			before, err := parsed.Data.Payload.Before.toEvent()
			if err != nil {
				return err
			}
			msgRequest.Payload.Before = *before
		}

		err = msgRequest.Payload.AddEventType(eventType)
		if err != nil {
			return err
		}
		return h.handlePaymentCreated(&msgRequest)
	}
}

func (h *PaymentHandler) handlePaymentCreated(paymentEvent *debezium.DebeziumMessage[domain.PaymentCreatedEvent]) error {
	// <-h.consumer.ReadyCH

	inv, inbox, err := inventory.PaymentToInventory(paymentEvent)
	if err != nil {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
		return err
	}

	inboxID, err := h.svc.Save(paymentEvent.Ctx, inbox, inv, paymentEvent.Metadata)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inbox.AggregateId,
				"OFFSET":      paymentEvent.Metadata.Offset,
				"PRTN":        paymentEvent.Metadata.Partition,
			},
		).Error("INSERT:ERROR")
		return err
	}
	logrus.WithFields(
		logrus.Fields{
			"inboxID":     inboxID,
			"aggregateID": inbox.AggregateId,
			"OFFSET":      paymentEvent.Metadata.Offset,
			"PRTN":        paymentEvent.Metadata.Partition,
		},
	).Info("INSERT:SUCCESS")
	return nil
}
