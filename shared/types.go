package shared

import (
	"github.com/google/uuid"
)

type Distance struct {
	// TODO -> linked list?
	// PrevID    string
	ID        string
	Value     float64
	Timestamp int64
}

type SensorData struct {
	SensorID uuid.UUID `json:"sensorID"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
}

type MsgBrokerType int

const (
	MsgBrokerType_Kafka MsgBrokerType = iota
)

type EventBusType int

const (
	EventBusType_InMemory EventBusType = iota
)
