package pkgconstants

type EventType string

const (
	EventType_PaymentCreated EventType = "payment_created"
	EventType_InboxCreated   EventType = "event_inbox_created"
)
