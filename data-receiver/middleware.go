package main

import (
	"github.com/k-code-yt/go-api-practice/shared"

	log "github.com/sirupsen/logrus"
)

type LogMiddleware struct {
	next DataProducer
}

func (l *LogMiddleware) ProduceData(data shared.SensorData) error {
	defer func() {
		log.WithFields(log.Fields{
			"sensorID": data.SensorID,
		}).Info("Producing")
	}()

	return l.next.ProduceData(data)
}

func NewLogMiddleware(next DataProducer) *LogMiddleware {
	return &LogMiddleware{
		next: next,
	}
}
