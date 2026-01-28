package handlers

// import (
// 	"context"
// 	"strconv"

// 	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
// 	application "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/application"
// 	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/payment/domain"
// 	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
// 	pkgkafka "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/kafka"
// 	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
// 	"github.com/sirupsen/logrus"
// )

// type CDCInventoryMsg struct {
// 	ID          int    `json:"id"`
// 	ProductName string `json:"product_name"`
// 	Status      string `json:"status"`
// 	Quantity    int    `json:"quantity"`
// 	LastUpdated int64  `json:"last_updated"`

// 	OrderNumber string `json:"order_number"`
// 	PaymentId   string `json:"payment_id"`
// }

// func (cdc *CDCInventoryMsg) toEvent() (*domain.InventoryCreatedEvent, error) {
// 	return &domain.InventoryCreatedEvent{
// 		ID:          cdc.ID,
// 		Status:      cdc.Status,
// 		ProductName: cdc.ProductName,
// 		Quantity:    cdc.Quantity,
// 		LastUpdated: debezium.ConvertIntTimeToUnix(cdc.LastUpdated),
// 		OrderNumber: cdc.OrderNumber,
// 		PaymentId:   cdc.PaymentId,
// 	}, nil
// }

// type InventoryHandler struct {
// 	Handler Handler
// 	svc     *application.PaymentService
// }

// func NewInventoryCreatedHandler(svc *application.PaymentService) *InventoryHandler {
// 	h := &InventoryHandler{
// 		svc: svc,
// 	}
// 	handlerFunc := h.CreateHandlerFunc()
// 	h.Handler = handlerFunc
// 	return h
// }

// func (h *InventoryHandler) CreateHandlerFunc() Handler {
// 	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition, eventType pkgtypes.EventType, cfg *pkgkafka.KafkaConfig) error {
// 		decoder := pkgkafka.GetDecoder[debezium.DebeziumMessage[CDCInventoryMsg]](cfg)
// 		parsed, err := pkgkafka.NewMessage(decoder, metadata, msg)
// 		if err != nil {
// 			return err
// 		}
// 		msgRequest := debezium.DebeziumMessage[domain.InventoryCreatedEvent]{
// 			Payload: debezium.Payload[domain.InventoryCreatedEvent]{
// 				Source:    parsed.Data.Payload.Source,
// 				Op:        parsed.Data.Payload.Op,
// 				Timestamp: parsed.Data.Payload.Timestamp,
// 				EventType: parsed.Data.Payload.EventType,
// 			},
// 			Metadata: metadata,
// 			Ctx:      ctx,
// 		}
// 		after, err := parsed.Data.Payload.After.toEvent()
// 		msgRequest.Payload.After = after
// 		if err != nil {
// 			return err
// 		}
// 		if parsed.Data.Payload.Before.ID != 0 {
// 			before, err := parsed.Data.Payload.Before.toEvent()
// 			if err != nil {
// 				return err
// 			}
// 			msgRequest.Payload.Before = before
// 		}

// 		err = msgRequest.Payload.AddEventType(eventType)
// 		if err != nil {
// 			return err
// 		}
// 		return h.handleInventoryCreateMsg(&msgRequest)
// 	}

// }

// func (h *InventoryHandler) handleInventoryCreateMsg(msg *debezium.DebeziumMessage[domain.InventoryCreatedEvent]) error {
// 	pID, err := strconv.Atoi(msg.Payload.After.PaymentId)
// 	if err != nil {
// 		return err
// 	}
// 	err = h.svc.Confirm(msg.Ctx, pID)
// 	if err != nil {
// 		return err
// 	}

// 	logrus.WithFields(
// 		logrus.Fields{
// 			"STATUS": "confirmed",
// 			"ID":     pID,
// 			"ORDER#": msg.Payload.After.OrderNumber,
// 		},
// 	).Info("PAYMENT:UPDATED")
// 	return nil
// }
