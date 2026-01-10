package service

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/debezium"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
)

func PaymentToInbox(p *debezium.DebeziumMessage[domain.Payment]) (*repo.InboxEvent, error) {
	afterJson, err := json.Marshal(p.Payload.After)
	if err != nil {
		return nil, err
	}
	return &repo.InboxEvent{
		Status:             repo.InboxEventStatus_Pending,
		InboxEventType:     p.Payload.EventType,
		AggregateId:        strconv.Itoa(p.Payload.After.ID),
		AggregateType:      "payment",
		AggregateMetadata:  afterJson,
		AggregateCreatedAt: p.Payload.After.CreatedAt,
		CreatedAt:          time.Now(),
	}, nil
}
