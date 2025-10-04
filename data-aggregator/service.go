package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	client "github.com/k-code-yt/go-api-practice/data-aggregator/transport"
	"github.com/k-code-yt/go-api-practice/shared"

	"github.com/sirupsen/logrus"
)

type Aggregator interface {
	AggregateDistance(context.Context, *shared.SensorData) error
	ProcessSensorDataForPayment(context.Context, *shared.SensorData) error
	GetDistance(id string) *shared.Distance
}

type AggregatorStorer interface {
	Insert(shared.Distance) (string, error)
	GetDistance(id string) *shared.Distance
}

type AggregatorService struct {
	points    [2][2]float64
	store     AggregatorStorer
	transport client.TransportClient
}

func NewAggregatorService(store AggregatorStorer, transport client.TransportClient) Aggregator {
	var points [2][2]float64
	aggServ := &AggregatorService{
		points:    points,
		store:     store,
		transport: transport,
	}

	return aggServ
}

func (svc *AggregatorService) AggregateDistance(ctx context.Context, data *shared.SensorData) error {
	if len(svc.points) == 0 {
		svc.points[0] = [2]float64{data.Lat, data.Lng}
		return nil
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
	return nil
}

func (svc *AggregatorService) ProcessSensorDataForPayment(ctx context.Context, d *shared.SensorData) error {
	logrus.Infof("ProcessSensorDataForPayment func got triggered with sensorID = %s\n", d.SensorID.String())
	errVal := rand.Intn(10)
	if errVal < 5 {
		return fmt.Errorf("random error for testing in ProcessSensorDataForPayment")
	}

	return nil
}

func (svc *AggregatorService) getDistanceBetweenPoints(p1, p2 [2]float64) float64 {
	return math.Sqrt(math.Pow(p1[0]-p2[0], 2) + math.Pow(p1[1]-p2[1], 2))
}

func (svc *AggregatorService) GetDistance(id string) *shared.Distance {
	return svc.store.GetDistance(id)
}
