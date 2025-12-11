package payment

import (
	"context"
	"fmt"

	paymentrepo "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/payment"
	reposhared "github.com/k-code-yt/go-api-practice/kafka-outbox/internal/repo/repo-shared"
)

type PaymentService struct {
	paymentRepo *paymentrepo.PaymentRepo
}

func NewPaymentService(pr *paymentrepo.PaymentRepo) *PaymentService {
	return &PaymentService{
		paymentRepo: pr,
	}
}

func (s *PaymentService) Save(ctx context.Context, p *paymentrepo.Payment) {
	id, err := s.paymentRepo.Save(ctx, p)
	if err != nil || id == reposhared.NonExistingIntKey {
		// TODO -> update status based on error or success
		fmt.Printf("ERR on DB SAVE = %v\n", err)
	}
	fmt.Printf("INSERT SUCCESS for PaymentID = %d\n", id)
}
