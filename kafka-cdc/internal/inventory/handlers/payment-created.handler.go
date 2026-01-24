package handlers

import (
	"fmt"
	"strconv"

	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	inventory "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/application"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	"github.com/sirupsen/logrus"
)

type CDCPayment struct {
	ID          int     `avro:"id"`
	OrderNumber string  `avro:"order_number"`
	Amount      string  `avro:"amount"`
	Status      *string `avro:"status"`
	CreatedAt   *int64  `avro:"created_at"`
	UpdatedAt   *int64  `avro:"updated_at"`
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
		Status:      *cdc.Status,
		CreatedAt:   debezium.ConvertIntTimeToUnix(*cdc.CreatedAt),
		UpdatedAt:   debezium.ConvertIntTimeToUnix(*cdc.UpdatedAt),
	}, nil
}

type PaymentHandler struct {
	// Handler Handler
	svc *application.InventoryService
}

func NewPaymentCreatedHandler(svc *application.InventoryService) *PaymentHandler {
	h := &PaymentHandler{
		svc: svc,
	}
	// handlerFunc := h.CreateHandlerFunc()
	// h.Handler = handlerFunc
	return h

}

// func getPaymentCreatedEvent[T any](envelope any) *domain.PaymentCreatedEvent {
// 	parsed := envelope.(T)

// 	msgRequest := debezium.DebeziumMessage[domain.PaymentCreatedEvent]{
// 		Payload: debezium.Payload[domain.PaymentCreatedEvent]{
// 			Source:    parsed.Data.Payload.Source,
// 			Op:        parsed.Data.Payload.Op,
// 			Timestamp: parsed.Data.Payload.Timestamp,
// 			EventType: parsed.Data.Payload.EventType,
// 		},
// 		Metadata: metadata,
// 		Ctx:      ctx,
// 	}
// 	after, err := parsed.Data.Payload.After.toEvent()
// 	msgRequest.Payload.After = after
// 	if err != nil {
// 		return err
// 	}
// 	if parsed.Data.Payload.Before.ID != 0 {
// 		before, err := parsed.Data.Payload.Before.toEvent()
// 		if err != nil {
// 			return err
// 		}
// 		msgRequest.Payload.Before = before
// 	}

// 	err = msgRequest.Payload.AddEventType(eventType)
// 	if err != nil {
// 		return err
// 	}

// }

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
