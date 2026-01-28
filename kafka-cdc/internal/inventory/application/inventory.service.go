package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	domain "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/infra/repo"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	pkgerrors "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/errors"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type InventoryService struct {
	inboxRepo     *repo.InboxEventRepo
	inventoryRepo *repo.InventoryRepo
}

func NewInventoryService(inboxRepo *repo.InboxEventRepo, invRepo *repo.InventoryRepo) *InventoryService {
	return &InventoryService{
		inboxRepo:     inboxRepo,
		inventoryRepo: invRepo,
	}
}

func (s *InventoryService) Save(ctx context.Context, inboxEvent *repo.InboxEvent, inv *domain.Inventory) (int, error) {
	txRepo := s.inboxRepo.GetRepo()
	id, err := postgres.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		logrus.WithFields(
			logrus.Fields{
				"aggregateID": inboxEvent.AggregateId,
			},
		).Info("INSERT:START")

		inboxID, err := s.inboxRepo.Insert(ctx, tx, inboxEvent)
		if err != nil {
			exists := postgres.IsDuplicateKeyErr(err)
			if exists {
				fmt.Printf("already exists AggregateID = %s\n", inboxEvent.AggregateId)
				return pkgerrors.CodeDuplicateKey, pkgerrors.NewDuplicateKeyError(err)
			}
			return pkgerrors.CodeNonExistingKey, pkgerrors.NewNonExistingKeyError(err)
		}
		s.inventoryRepo.Insert(ctx, tx, inv)

		logrus.WithFields(
			logrus.Fields{
				"eventID":     inboxID,
				"aggregateID": inboxEvent.AggregateId,
			},
		).Info("INSERT:SUCCESS")
		return inboxID, nil
	})

	return id, err
}

func PaymentToInventory(payment *domain.PaymentCreatedEvent, eventType pkgtypes.EventType) (*domain.Inventory, *repo.InboxEvent, error) {
	afterJson, err := json.Marshal(payment)
	if err != nil {
		return nil, nil, err
	}
	inbox := &repo.InboxEvent{
		Status:             repo.InboxEventStatus_Pending,
		InboxEventType:     eventType,
		AggregateId:        strconv.Itoa(payment.ID),
		AggregateType:      "payment",
		AggregateMetadata:  afterJson,
		AggregateCreatedAt: payment.CreatedAt,
		CreatedAt:          time.Now(),
	}
	inv := domain.NewInventoryReservation(payment.OrderNumber, strconv.Itoa(payment.ID), int(payment.Amount))
	return inv, inbox, err

}
