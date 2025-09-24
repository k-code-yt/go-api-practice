package main

import (
	"go-logistic-api/shared"
	"math"

	"github.com/sirupsen/logrus"
)

type AggregatorSerivce struct {
	dataCH    chan shared.SensorData
	points    [2][2]float64
	distances []float64
}

func NewAggregatorService() *AggregatorSerivce {
	var points [2][2]float64
	return &AggregatorSerivce{
		dataCH: make(chan shared.SensorData, 128),
		points: points,
	}
}

func (as *AggregatorSerivce) ReadMessageLoop() {
	for {
		select {
		case v := <-as.dataCH:
			as.calcDistance(v)
		}

	}
}

func (as *AggregatorSerivce) calcDistance(data shared.SensorData) {
	if len(as.points) == 0 {
		as.points[0] = [2]float64{data.Lat, data.Lng}
		return
	}

	as.points[1] = as.points[0]
	as.points[0] = [2]float64{data.Lat, data.Lng}

	distance := as.getDistanceBetweenPoints(as.points[0], as.points[1])
	as.distances = append(as.distances, distance)
	logrus.Infof("distance = %f\n", as.distances[len(as.distances)-1])
}

func (as *AggregatorSerivce) getDistanceBetweenPoints(p1, p2 [2]float64) float64 {
	return math.Sqrt(math.Pow(p1[0]-p2[0], 2) + math.Pow(p1[1]-p2[1], 2))
}
