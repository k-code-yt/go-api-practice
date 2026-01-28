package handlers

import (
	"context"

	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	pkgtypes "github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/types"
)

type Handlers interface {
	HandlePaymentCreated(ctx context.Context, paymentEvent *domain.PaymentCreatedEvent, eventType pkgtypes.EventType) error
}
