package service

import (
	"encoding/json"
	"strconv"

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
		InboxEventType: p.Payload.EventType,
		CreatedAt:      p.Payload.After.CreatedAt,
		Status:         repo.InboxEventStatus_Pending,
		ParentId:       strconv.Itoa(p.Payload.After.ID),
		ParentType:     "payment",
		ParentMetadata: afterJson,
	}, nil
}
