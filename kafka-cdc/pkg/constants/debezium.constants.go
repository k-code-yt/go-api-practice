package pkgconstants

import "fmt"

const (
	DebConnectorName       = "audit_cdc"
	CDCPaymentTableName    = "payment"
	CDCInboxEventTableName = DBTableName_InboxEvents
)

var (
	DebDefaultTopic = fmt.Sprintf("cdc.public.%s", CDCPaymentTableName)
	DebInboxTopic   = fmt.Sprintf("cdc.public.%s", CDCInboxEventTableName)
)
