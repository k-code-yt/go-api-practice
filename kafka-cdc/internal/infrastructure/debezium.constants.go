package infrastructure

import (
	"fmt"

	invconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/inventory/constants"
	paymconstants "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/infrastructure/payment/constants"
)

const (
	CDCPaymentTableName   = paymconstants.DBTableName_Payment
	CDCInventoryTableName = invconstants.DBTableName_Inventory
)

var (
	DebInventoryTopic = fmt.Sprintf("cdc.public.%s", CDCInventoryTableName)
	DebPaymentTopic   = fmt.Sprintf("cdc.public.%s", CDCPaymentTableName)
)
