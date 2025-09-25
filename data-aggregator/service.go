package main

import (
	"go-logistic-api/shared"
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

type Aggregator interface {
	AggregateDistance(shared.SensorData)
	GetDistance(id string) *shared.Distance
}

type AggregatorStorer interface {
	Insert(shared.Distance) (string, error)
	GetDistance(id string) *shared.Distance
}

type AggregatorService struct {
	points   [2][2]float64
	store    AggregatorStorer
	eventBus AggregatorEventBus
}

func NewAggregatorService(store AggregatorStorer, eb AggregatorEventBus) Aggregator {
	var points [2][2]float64
	aggServ := &AggregatorService{
		points:   points,
		store:    store,
		eventBus: eb,
	}

	return aggServ
}

func (as *AggregatorService) AggregateDistance(data shared.SensorData) {
	if len(as.points) == 0 {
		as.points[0] = [2]float64{data.Lat, data.Lng}
		return
	}

	as.points[1] = as.points[0]
	as.points[0] = [2]float64{data.Lat, data.Lng}

	distance := as.getDistanceBetweenPoints(as.points[0], as.points[1])
	d := shared.Distance{
		Value:     distance,
		Timestamp: time.Now().Unix(),
	}
	_, err := as.store.Insert(d)
	if err != nil {
		logrus.Error("Error saving data ", err)
	}
}

func (as *AggregatorService) getDistanceBetweenPoints(p1, p2 [2]float64) float64 {
	return math.Sqrt(math.Pow(p1[0]-p2[0], 2) + math.Pow(p1[1]-p2[1], 2))
}

func (as *AggregatorService) GetDistance(id string) *shared.Distance {
	return as.store.GetDistance(id)
}
