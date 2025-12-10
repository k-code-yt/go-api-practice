package payment

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/event"
	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/payment"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/repo-shared"
)

type PaymentService struct {
	paymentRepo *paymentrepo.PaymentRepo
}

func NewPaymentService(db *sqlx.DB) *PaymentService {
	eventRepo := event.NewEventRepo(db)
	pr := paymentrepo.NewPaymentRepo(db, eventRepo)
	return &PaymentService{
		paymentRepo: pr,
	}
}

func (s *PaymentService) Save(ctx context.Context, p *paymentrepo.Payment) {
	id, err := s.paymentRepo.Save(ctx, p)
	if err != nil || id == reposhared.NonExistingIntKey {
		// TODO -> update status based on error or success
		// fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	// fmt.Printf("INSERT SUCCESS for OFFSET = %d, PRTN = %d, EventID = %s\n", msg.Metadata.Offset, msg.Metadata.Partition, id)

}
