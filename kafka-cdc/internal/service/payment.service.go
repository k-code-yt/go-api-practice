package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	dbpostgres "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/db/postgres"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos/repo-shared"
	"github.com/sirupsen/logrus"
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

func (pr *PaymentService) Save(ctx context.Context, p *domain.Payment) (int, error) {
	txRepo := pr.eventRepo.GetRepo()
	id, err := reposhared.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		fmt.Printf("starting DB operation for order# = %s\n", p.OrderNumber)

		paymentID, err := pr.paymentRepo.Insert(ctx, tx, p)
		if err != nil {
			return dbpostgres.NonExistingIntKey, err
		}
		p.ID = paymentID
		logrus.WithFields(
			logrus.Fields{
				"paymentID": paymentID,
				"order#":    p.OrderNumber,
			},
		).Info("INSERT SUCCESS")
		return paymentID, nil
	})

	if err != nil || id == dbpostgres.NonExistingIntKey {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	return id, err
}

func (pr *PaymentService) Confirm(ctx context.Context, ID int) error {
	txRepo := pr.eventRepo.GetRepo()
	_, err := reposhared.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		return 0, pr.paymentRepo.UpdateStatus(ctx, tx, ID, "confirmed")
	})
	if err != nil {
		fmt.Printf("ERR on DB STATUS UPDATE = %v\n", err)
	}
	return err
}
