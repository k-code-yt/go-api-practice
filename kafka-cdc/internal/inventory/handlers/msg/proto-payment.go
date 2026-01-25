package handlersmsg

import (
	"context"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/debezium"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type ProtoMapper struct {
}

func NewProtoMapper() *ProtoMapper {
	return &ProtoMapper{}
}

func (m *ProtoMapper) paymentToEvent(cdc *pkgtypes.CDCPayment) (*domain.PaymentCreatedEvent, error) {
	amount, err := strconv.ParseFloat(cdc.Amount, 2)
	if err != nil {
		amount, err = debezium.DecodeDebeziumDecimal(cdc.Amount, 2)
		if err != nil {
			return nil, err
		}
	}

	return &domain.PaymentCreatedEvent{
		ID:          int(cdc.Id),
		OrderNumber: cdc.OrderNumber,
		Amount:      amount,
		Status:      cdc.Status,
		CreatedAt:   debezium.ConvertIntTimeToUnix(cdc.CreatedAt),
		UpdatedAt:   debezium.ConvertIntTimeToUnix(cdc.UpdatedAt),
	}, nil
}

func (m *ProtoMapper) getPaymentCreatedEvent(ctx context.Context, envelope any, eventType pkgtypes.EventType, metadata *kafka.TopicPartition) (*debezium.DebeziumMessage[domain.PaymentCreatedEvent], error) {
	parsed := envelope.(*pkgtypes.CDCPaymentEnvelope)

	msgRequest := debezium.DebeziumMessage[domain.PaymentCreatedEvent]{
		Payload: debezium.Payload[domain.PaymentCreatedEvent]{
			// TODO -> add mapping
			// Source:    parsed.Source,
			Op:        parsed.Op,
			Timestamp: &parsed.TsMs,
			EventType: eventType,
		},
		Metadata: metadata,
		Ctx:      ctx,
	}
	after, err := m.paymentToEvent(parsed.After)
	msgRequest.Payload.After = after
	if err != nil {
		return nil, err
	}
	if parsed.Before != nil {
		before, err := m.paymentToEvent(parsed.Before)
		if err != nil {
			return nil, err
		}
		msgRequest.Payload.Before = before
	}

	err = msgRequest.Payload.AddEventType(eventType)
	if err != nil {
		return nil, err
	}
	return &msgRequest, nil
}
