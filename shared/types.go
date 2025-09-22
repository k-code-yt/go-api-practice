package shared

import (
	"github.com/google/uuid"
)

type SensorData struct {
	SensorID uuid.UUID `json:"sensorID"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
}
