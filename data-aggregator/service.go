package main

import (
	"fmt"
	client "go-logistic-api/data-aggregator/transport"
	"go-logistic-api/shared"
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

type Aggregator interface {
	AggregateDistance(*shared.SensorData)
	GetDistance(id string) *shared.Distance
	AcceptSensorData(*shared.SensorData)
}

type AggregatorStorer interface {
	Insert(shared.Distance) (string, error)
	GetDistance(id string) *shared.Distance
}

type AggregatorService struct {
	points    [2][2]float64
	store     AggregatorStorer
	eventBus  AggregatorEventBus
	transport client.TransportClient
}

func NewAggregatorService(store AggregatorStorer, eb AggregatorEventBus) Aggregator {
	var points [2][2]float64
	aggServ := &AggregatorService{
		points:    points,
		store:     store,
		eventBus:  eb,
		transport: client.NewHTTPClient(fmt.Sprintf("http://localhost%s/invoice", shared.HTTPPortInvoice)),
	}

	return aggServ
}

func (svc *AggregatorService) AggregateDistance(data *shared.SensorData) {
	if len(svc.points) == 0 {
		svc.points[0] = [2]float64{data.Lat, data.Lng}
		return
	}

	svc.points[1] = svc.points[0]
	svc.points[0] = [2]float64{data.Lat, data.Lng}

	distance := svc.getDistanceBetweenPoints(svc.points[0], svc.points[1])
	d := shared.Distance{
		Value:     distance,
		Timestamp: time.Now().Unix(),
	}
	_, err := svc.store.Insert(d)
	if err != nil {
		logrus.Error("Error saving data ", err)
	}

	svc.transport.SaveInvoice(d)
}

func (svc *AggregatorService) AcceptSensorData(d *shared.SensorData) {
	logrus.Infof("Second subsciber func got triggered with sensorID = %s\n", d.SensorID.String())
}

func (svc *AggregatorService) getDistanceBetweenPoints(p1, p2 [2]float64) float64 {
	return math.Sqrt(math.Pow(p1[0]-p2[0], 2) + math.Pow(p1[1]-p2[1], 2))
}

func (svc *AggregatorService) GetDistance(id string) *shared.Distance {
	return svc.store.GetDistance(id)
}
