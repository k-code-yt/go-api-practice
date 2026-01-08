package debezium

import (
	"fmt"
	"strings"
)

type DebeziumMessage[T any] struct {
	Payload Payload[T]
	// Schema  json.RawMessage
}

type Payload[T any] struct {
	Before    T
	After     T
	Source    Source
	Op        string `json:"op"`
	Timestamp int    `json:"ts_ms"`
	EventType string
}

type Source struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	Timestamp int    `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	Sequence  string `json:"sequence"`
	Table     string `json:"table"`
	TxId      int    `json:"txId"`
	Lsn       int    `json:"lsn"`
}

func (p *Payload[T]) AddEventType(topic string) error {
	r := strings.Split(topic, ".")
	if topic == "" || len(r) < 3 {
		return fmt.Errorf("Invalid topic & event type combination")
	}

	et := r[2]
	switch p.Op {
	case "c", "r":
		et = fmt.Sprintf("%s_%s", et, "created")
	case "u":
		et = fmt.Sprintf("%s_%s", et, "updated")
	}

	p.EventType = et
	return nil
}
