package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos/repo-shared"
	"github.com/sirupsen/logrus"
)

type InboxService struct {
	inboxRepo *repo.InboxEventRepo
	consumer  *consumer.KafkaConsumer
}

func NewInboxService(ier *repo.InboxEventRepo) *InboxService {
	return &InboxService{
		inboxRepo: ier,
	}
}

func (s *InboxService) AddConsumer(consumer *consumer.KafkaConsumer) {
	s.consumer = consumer
}

func (s *InboxService) Save(ctx context.Context, inboxEvent *repo.InboxEvent, metadata *kafka.TopicPartition) (int, error) {
	txRepo := s.inboxRepo.GetRepo()
	id, err := reposhared.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		logrus.WithFields(
			logrus.Fields{
				"OFFSET":      metadata.Offset,
				"PRTN":        metadata.Partition,
				"aggregateID": inboxEvent.AggregateId,
			},
		).Info("INSERT:START")

		inboxID, err := s.inboxRepo.Insert(ctx, tx, inboxEvent)
		if err != nil {
			exists := dbpostgres.IsDuplicateKeyErr(err)
			if exists {
				eMsg := fmt.Sprintf("already exists ID = %d, PRTN = %d, AggregateID = %s\n", metadata.Offset, metadata.Partition, inboxEvent.AggregateId)
				s.consumer.UpdateState(metadata, consumer.MsgState_Success)
				return dbpostgres.DuplicateKeyViolation, errors.New(eMsg)
			}
			// TODO -> add DLQ handling
			s.consumer.UpdateState(metadata, consumer.MsgState_Error)
			return dbpostgres.NonExistingIntKey, err
		}
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
