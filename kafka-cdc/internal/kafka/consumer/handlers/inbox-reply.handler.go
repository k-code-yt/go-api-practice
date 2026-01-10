package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	pkgconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/constants"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type CDCInboxMsg struct {
	ID             int                   `json:"id"`
	InboxEventType string                `json:"type"`
	AggregateId    string                `json:"aggregate_id"`
	AggregateType  string                `json:"aggregate_type"`
	Status         repo.InboxEventStatus `json:"status"`
}

func (cdc *CDCInboxMsg) toDomain() (*repo.InboxEvent, error) {
	return &repo.InboxEvent{
		ID:             cdc.ID,
		AggregateId:    cdc.AggregateId,
		InboxEventType: pkgconstants.EventType(cdc.InboxEventType),
		AggregateType:  repo.InboxEventParentType(cdc.AggregateType),
		Status:         cdc.Status,
	}, nil
}

type InboxReplyHandler struct {
	Handler Handler
	MsgCH   chan *debezium.DebeziumMessage[repo.InboxEvent]
	timeout time.Duration
}

func NewInboxReplyCreatedHandler() *InboxReplyHandler {
	h := &InboxReplyHandler{
		MsgCH:   make(chan *debezium.DebeziumMessage[repo.InboxEvent], 64),
		timeout: time.Second * 10,
	}
	handlerFunc := h.CreateHandlerFunc()
	h.Handler = handlerFunc
	return h

}

func (h *InboxReplyHandler) CreateHandlerFunc() Handler {
	return func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error {
		parsed, err := pkgtypes.NewMessage[debezium.DebeziumMessage[CDCInboxMsg]](metadata, msg)
		if err != nil {
			return err
		}
		msgRequest := debezium.DebeziumMessage[repo.InboxEvent]{
			Payload: debezium.Payload[repo.InboxEvent]{
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
