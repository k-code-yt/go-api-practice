package pkgconstants

import "fmt"

const (
	CDCPaymentTableName  = DBTableName_Payment
	CDCInvntoryTableName = DBTableName_Inventory
)

var (
	DebPaymentTopic             = fmt.Sprintf("cdc.public.%s", CDCPaymentTableName)
	DebInventoryTopic           = fmt.Sprintf("cdc.public.%s", DBTableName_Inventory)
	DebInventoryDBConnectorName = fmt.Sprintf("%s_conn", DBNameInventory)
	DebPaymentDBConnectorName   = fmt.Sprintf("%s_conn", DBNamePrimary)
)
