package msg

import (
	"fmt"
)

const (
	CDCPaymentTableName = "payment"
)

var (
	DebPaymentTopic = fmt.Sprintf("cdc.public.%s", CDCPaymentTableName)
)
