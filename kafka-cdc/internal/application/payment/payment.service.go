package payment

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/domain/payment"
	outboxrepo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/outbox"
	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/payment"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
	"github.com/sirupsen/logrus"
)

type PaymentService struct {
	paymentRepo *paymentrepo.PaymentRepo
	eventRepo   *outboxrepo.EventRepo
}

func NewPaymentService(pr *paymentrepo.PaymentRepo, er *outboxrepo.EventRepo) *PaymentService {
	return &PaymentService{
		paymentRepo: pr,
		eventRepo:   er,
	}
}

func (pr *PaymentService) Save(ctx context.Context, p *payment.Payment) (int, error) {
	txRepo := pr.eventRepo.GetRepo()
	id, err := postgres.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		fmt.Printf("starting DB operation for order# = %s\n", p.OrderNumber)

		paymentID, err := pr.paymentRepo.Insert(ctx, tx, p)
		if err != nil {
			return postgres.NonExistingIntKey, err
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

	if err != nil || id == postgres.NonExistingIntKey {
		fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	return id, err
}

func (pr *PaymentService) Confirm(ctx context.Context, ID int) error {
	txRepo := pr.eventRepo.GetRepo()
	_, err := postgres.TxClosure(ctx, txRepo, func(ctx context.Context, tx *sqlx.Tx) (int, error) {
		return 0, pr.paymentRepo.UpdateStatus(ctx, tx, ID, "confirmed")
	})
	if err != nil {
		fmt.Printf("ERR on DB STATUS UPDATE = %v\n", err)
	}
	return err
}
