package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
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

func (cdc *CDCPayment) toDomain() (*domain.Payment, error) {
	amount, err := strconv.ParseFloat(cdc.Amount, 2)
	if err != nil {
		amount, err = decodeDebeziumDecimal(cdc.Amount, 2)
		if err != nil {
			return nil, err
		}
	}
	return &domain.Payment{
		ID:          cdc.ID,
		OrderNumber: cdc.OrderNumber,
		Amount:      amount,
		Status:      cdc.Status,
		CreatedAt:   convertIntTimeToUnix(cdc.CreatedAt),
		UpdatedAt:   convertIntTimeToUnix(cdc.UpdatedAt),
	}, nil
}

func convertIntTimeToUnix(microSeconds int64) time.Time {
	seconds := microSeconds / 1_000_000
	nanos := (microSeconds % 1_000_000) * 1000
	return time.Unix(int64(seconds), int64(nanos))
}

type PaymentCreatedHandler struct {
	Handler Handler
	MsgCH   chan *debezium.DebeziumMessage[domain.Payment]
	timeout time.Duration
}

func NewPaymentCreatedHandler() *PaymentCreatedHandler {
	h := &PaymentCreatedHandler{
		MsgCH:   make(chan *debezium.DebeziumMessage[domain.Payment], 64),
		timeout: time.Second * 10,
	}
	handlerFunc := h.CreateHandlerFunc()
	h.Handler = handlerFunc
	return h

}

func (h *PaymentCreatedHandler) CreateHandlerFunc() Handler {
	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error {
		parsed, err := pkgtypes.NewMessage[debezium.DebeziumMessage[CDCPayment]](metadata, msg)
		if err != nil {
			return err
		}
		msgRequest := debezium.DebeziumMessage[domain.Payment]{
			Payload: debezium.Payload[domain.Payment]{
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
