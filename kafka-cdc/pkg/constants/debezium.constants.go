package pkgconstants

import "fmt"

const (
	DebConnectorName    = "audit_cdc"
	CDCPaymentTableName = "payment"
)

var (
	DebDefaultTopic = fmt.Sprintf("cdc.public.%s", CDCPaymentTableName)
)
