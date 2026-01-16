package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type CDCInventoryMsg struct {
	ID          int    `db:"id"`
	ProductName string `db:"product_name"`
	Status      string `db:"status"`
	Quantity    int    `db:"quantity"`
	LastUpdated int64  `db:"last_updated"`

	OrderNumber string `db:"order_number"`
	PaymentId   string `db:"payment_id"`
}

func (cdc *CDCInventoryMsg) toDomain() (*domain.Inventory, error) {
	return &domain.Inventory{
		ID:          cdc.ID,
		Status:      cdc.Status,
		ProductName: cdc.ProductName,
		Quantity:    cdc.Quantity,
		LastUpdated: convertIntTimeToUnix(cdc.LastUpdated),
		OrderNumber: cdc.OrderNumber,
		PaymentId:   cdc.PaymentId,
	}, nil
}

type InventoryHandler struct {
	Handler Handler
	MsgCH   chan *debezium.DebeziumMessage[domain.Inventory]
	timeout time.Duration
}

func NewInventoryCreatedHandler() *InventoryHandler {
	h := &InventoryHandler{
		MsgCH:   make(chan *debezium.DebeziumMessage[domain.Inventory], 64),
		timeout: time.Second * 10,
	}
	handlerFunc := h.CreateHandlerFunc()
	h.Handler = handlerFunc
	return h

}

func (h *InventoryHandler) CreateHandlerFunc() Handler {
	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error {
		parsed, err := pkgtypes.NewMessage[debezium.DebeziumMessage[CDCInventoryMsg]](metadata, msg)
		if err != nil {
			return err
		}
		msgRequest := debezium.DebeziumMessage[domain.Inventory]{
			Payload: debezium.Payload[domain.Inventory]{
				Source:    parsed.Data.Payload.Source,
				Op:        parsed.Data.Payload.Op,
				Timestamp: parsed.Data.Payload.Timestamp,
				EventType: parsed.Data.Payload.EventType,
			},
			Metadata: metadata,
			Ctx:      ctx,
		}
		after, err := parsed.Data.Payload.After.toDomain()
		msgRequest.Payload.After = *after
		if err != nil {
			return err
		}
		if parsed.Data.Payload.Before.ID != 0 {
			before, err := parsed.Data.Payload.Before.toDomain()
			if err != nil {
				return err
			}
			msgRequest.Payload.Before = *before
		}

		err = msgRequest.Payload.AddEventType(*metadata.Topic)
		if err != nil {
			return err
		}

		select {
		case h.MsgCH <- &msgRequest:

		case <-time.After(h.timeout):
			logrus.Errorf("MsgCH blocked for %fs, dropping message offset=%d partition=%d",
				h.timeout.Seconds(), metadata.Offset, metadata.Partition)
			return fmt.Errorf("timeout exceeded, marking offset as errored")
		}

		return nil
	}

}
