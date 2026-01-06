package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/db/postgres"
	repo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repos"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repos/repo-shared"
)

type PaymentService struct {
	paymentRepo *repo.PaymentRepo
	eventRepo   *repo.EventRepo
}

func NewPaymentService(pr *repo.PaymentRepo, er *repo.EventRepo) *PaymentService {
	return &PaymentService{
		paymentRepo: pr,
		eventRepo:   er,
	}
}

func (pr *PaymentService) Save(ctx context.Context, p *repo.Payment) (int, error) {
	txRepo := pr.eventRepo.GetRepo()
	id, err := reposhared.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		// fmt.Printf("starting DB operation for order# = %s\n", p.OrderNumber)

		paymentID, err := pr.paymentRepo.Insert(ctx, tx, p)
		if err != nil {
			return dbpostgres.NonExistingIntKey, err
		}
		p.ID = paymentID
		metadata, err := json.Marshal(p)
		if err != nil {
			return dbpostgres.NonExistingIntKey, err
		}

		event := repo.NewEvent(repo.EventType_PaymentCreated, strconv.Itoa(paymentID), repo.EventParentType_Payment, metadata)
		_, err = pr.eventRepo.Insert(ctx, tx, event)
		if err != nil {
			return dbpostgres.NonExistingIntKey, err
		}
		// logrus.WithFields(
		// 	logrus.Fields{
		// 		"eventID":   eventID,
		// 		"paymentID": paymentID,
		// 		"order#":    p.OrderNumber,
		// 	},
		// ).Info("INSERT SUCCESS")
		return paymentID, nil
	})

	if err != nil || id == dbpostgres.NonExistingIntKey {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	return id, err
}
