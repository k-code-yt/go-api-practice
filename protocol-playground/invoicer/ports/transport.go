package ports

type ServerTransport interface {
	Listen(svc Invoicer) error
}
