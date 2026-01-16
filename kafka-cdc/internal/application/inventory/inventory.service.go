package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/inventory"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	"github.com/sirupsen/logrus"
)

type InventoryService struct {
	inboxRepo     *repo.InboxEventRepo
	inventoryRepo *repo.InventoryRepo
	consumer      *consumer.KafkaConsumer
}

func NewInventoryService(inboxRepo *repo.InboxEventRepo, invRepo *repo.InventoryRepo) *InventoryService {
	return &InventoryService{
		inboxRepo:     inboxRepo,
		inventoryRepo: invRepo,
	}
}

func (s *InventoryService) AddConsumer(consumer *consumer.KafkaConsumer) {
	s.consumer = consumer
}

func (s *InventoryService) Save(ctx context.Context, inboxEvent *repo.InboxEvent, inv *inventory.Inventory, metadata *kafka.TopicPartition) (int, error) {
	txRepo := s.inboxRepo.GetRepo()
	id, err := postgres.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		logrus.WithFields(
			logrus.Fields{
				"OFFSET":      metadata.Offset,
				"PRTN":        metadata.Partition,
				"aggregateID": inboxEvent.AggregateId,
			},
		).Info("INSERT:START")

		inboxID, err := s.inboxRepo.Insert(ctx, tx, inboxEvent)
		if err != nil {
			exists := postgres.IsDuplicateKeyErr(err)
			if exists {
				eMsg := fmt.Sprintf("already exists ID = %d, PRTN = %d, AggregateID = %s\n", metadata.Offset, metadata.Partition, inboxEvent.AggregateId)
				s.consumer.UpdateState(metadata, consumer.MsgState_Success)
				return postgres.DuplicateKeyViolation, errors.New(eMsg)
			}
			// TODO -> add DLQ handling
			s.consumer.UpdateState(metadata, consumer.MsgState_Error)
			return postgres.NonExistingIntKey, err
		}
		s.inventoryRepo.Insert(ctx, tx, inv)

		s.consumer.UpdateState(metadata, consumer.MsgState_Success)
		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inboxEvent.AggregateId,
				"OFFSET":      metadata.Offset,
				"PRTN":        metadata.Partition,
			},
		).Info("INSERT:SUCCESS")
		return inboxID, nil
	})

	return id, err
}

func PaymentToInventory(p *infrastructure.DebeziumMessage[payment.Payment]) (*inventory.Inventory, *repo.InboxEvent, error) {
	payment := p.Payload.After
	afterJson, err := json.Marshal(payment)
	if err != nil {
		return nil, nil, err
	}
	inbox := &repo.InboxEvent{
		Status:             repo.InboxEventStatus_Pending,
		InboxEventType:     p.Payload.EventType,
		AggregateId:        strconv.Itoa(payment.ID),
		AggregateType:      "payment",
		AggregateMetadata:  afterJson,
		AggregateCreatedAt: payment.CreatedAt,
		CreatedAt:          time.Now(),
	}
	inv := inventory.NewInventoryReservation(payment.OrderNumber, strconv.Itoa(payment.ID), int(payment.Amount))
	return inv, inbox, err

}
