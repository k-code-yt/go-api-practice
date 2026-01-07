package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/kafka/consumer"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos/repo-shared"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
	"github.com/sirupsen/logrus"
)

type InboxService struct {
	inboxRepo *repo.InboxEventRepo
	consumer  *consumer.KafkaConsumer[repo.Event]
}

func NewInboxService(ier *repo.InboxEventRepo) *InboxService {
	return &InboxService{
		inboxRepo: ier,
	}
}

func (s *InboxService) AddConsumer(consumer *consumer.KafkaConsumer[repo.Event]) {
	s.consumer = consumer
}

func (s *InboxService) Save(ctx context.Context, msg *pkgtypes.Message[repo.Event]) (int, error) {
	txRepo := s.inboxRepo.GetRepo()
	id, err := reposhared.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		// fmt.Printf("starting DB operation for offset = %s\n", msg.Metadata.Offset)
		inboxEvent := repo.NewInboxEvent(&msg.Data)

		inboxID, err := s.inboxRepo.Insert(ctx, tx, inboxEvent)

		// --- BL---
		// 1. do BL here -> longer commits to kafka
		// 2. many service -> chan || cron by status "pending"
		//  	go routine -> service factory
		//  faster -> save event -> one by one seq.
		if err != nil {
			exists := dbpostgres.IsDuplicateKeyErr(err)
			if exists {
				eMsg := fmt.Sprintf("already exists OFFSET = %d, PRTN = %d, OutboxEventID = %s\n", msg.Metadata.Offset, msg.Metadata.Partition, msg.Data.EventId)
				s.consumer.UpdateState(msg.Metadata, consumer.MsgState_Success)
				return dbpostgres.NonExistingIntKey, errors.New(eMsg)
			}
			// TODO -> add DLQ handling
			s.consumer.UpdateState(msg.Metadata, consumer.MsgState_Error)
			return dbpostgres.NonExistingIntKey, err
		}
		s.consumer.UpdateState(msg.Metadata, consumer.MsgState_Success)
		logrus.WithFields(
			logrus.Fields{
				"eventID":       inboxID,
				"outboxEventID": msg.Data.EventId,
			},
		).Info("INSERT SUCCESS")
		return inboxID, nil
	})

	if err != nil || id == dbpostgres.NonExistingIntKey {
		// fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	return id, err
}
