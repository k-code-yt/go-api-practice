package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	grpcclient "github.com/k-code-yt/go-api-practice/data-aggregator/transport/grpc"
	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	go signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	eventBus := EventBusFactory[*shared.SensorData](EventBusConfig{
		eventBusType: shared.EventBusType_InMemory,
	})

	go func() {
		fmt.Println("waiting for SIGTERM")
		<-sigChan
		eventBus.Close()
	}()

	msgBroker, err := NewMsgBroker(&MsgBrokerConfig{
		brokerType: shared.MsgBrokerType_Kafka,
		eb:         eventBus,
	})
	if err != nil {
		panic(err)
	}

	aggrStore := NewInMemoryStore()
	dataCH := make(chan *shared.SensorDataProto, 128)
	integrationTransport, err := grpcclient.NewGRPCClient(fmt.Sprintf("localhost%s", shared.HTTPPortInvoice), dataCH)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	go integrationTransport.AcceptWSLoop()
	go integrationTransport.ReadServerLoop()

	aggrService := NewAggregatorService(aggrStore, integrationTransport)

	go msgBroker.consumer.ReadMessageLoop()
	eventBus.Subscribe(shared.CalculatePaymentDomainEvent, aggrService.ProcessSensorDataForPayment)
	eventBus.Subscribe(shared.AggregateDistanceDomainEvent, aggrService.AggregateDistance)

	http.HandleFunc("/ws", handleWS(dataCH))
	http.HandleFunc("/get-distance", handleGetDistance(aggrService))
	http.HandleFunc("/get-invoice", handleGetInvoice(integrationTransport))
	logrus.Infof("Starting Aggregator HTTP listener at: %s\n", shared.HTTPPortAggregator)
	log.Fatal(http.ListenAndServe(shared.HTTPPortAggregator, nil))
}
