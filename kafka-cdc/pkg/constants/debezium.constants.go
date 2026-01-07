package pkgconstants

import "fmt"

const (
	DebConnectorName = "audit_cdc"
	DebDefaultTable  = DBTableName_Events
)

var (
	DebDefaultTopic = fmt.Sprintf("%s.payment_created", DebDefaultTable)
)
