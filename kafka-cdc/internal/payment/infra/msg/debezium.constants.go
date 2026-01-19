package msg

import (
	"fmt"
)

const (
	CDCInventoryTableName = "inventory"
)

var (
	DebInventoryTopic = fmt.Sprintf("cdc.public.%s", CDCInventoryTableName)
)
