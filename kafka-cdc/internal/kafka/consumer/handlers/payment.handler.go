package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type CDCPayment struct {
	ID          int       `db:"id"`
	OrderNumber string    `db:"order_number"`
	Amount      string    `db:"amount"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (cdc *CDCPayment) toDomain() (*domain.Payment, error) {
	amount, err := strconv.Atoi(cdc.Amount)
	if err != nil {
		// amount, err := decodeDebeziumDecimal(cdc.Amount)
		if err != nil {
			return nil, err
		}
	}
	return &domain.Payment{
		ID:          cdc.ID,
		OrderNumber: cdc.OrderNumber,
		Amount:      amount,
		Status:      cdc.Status,
		CreatedAt:   cdc.CreatedAt,
		UpdatedAt:   cdc.UpdatedAt,
	}, nil
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

func decodeDebeziumDecimal(encoded string, scale int) (float64, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, err
	}

	bigInt := new(big.Int).SetBytes(decoded)

	bigFloat := new(big.Float).SetInt(bigInt)

	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	result := new(big.Float).Quo(bigFloat, divisor)

	val, _ := result.Float64()
	return val, nil
}
