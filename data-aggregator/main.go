package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	grpcclient "github.com/k-code-yt/go-api-practice/data-aggregator/transport/grpc"
	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
)

func handleGetDistance(svc Aggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		id := queryParams.Get("id")
		d := svc.GetDistance(id)
		err := writeJSON(w, http.StatusOK, d)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "request failed, unable to marshal payload"})
			return
		}
	}
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "applicaiton/json")
	return json.NewEncoder(rw).Encode(v)
}

func main() {
	eventBus := EventBusFactory(EventBusConfig{
		eventBusType: shared.EventBusType_InMemory,
	})
	msgBroker, err := NewMsgBroker(&MsgBrokerConfig{
		brokerType: shared.MsgBrokerType_Kafka,
		eb:         eventBus,
	})
	if err != nil {
		panic(err)
	}

	aggrStore := NewInMemoryStore()
	transport, err := grpcclient.NewGRPCClient(fmt.Sprintf("localhost%s", shared.HTTPPortInvoice))
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	aggrService := NewAggregatorService(aggrStore, eventBus, transport)

	eventBus.CreateTopic("invoice-calculator")
	eventBus.CreateTopic("distance-calculator")
	go msgBroker.consumer.ReadMessageLoop()
	go eventBus.Subscribe("invoice-calculator", aggrService.AcceptSensorData)
	go eventBus.Subscribe("distance-calculator", aggrService.AggregateDistance)

	http.HandleFunc("/get-distance", handleGetDistance(aggrService))
	log.Fatal(http.ListenAndServe(shared.HTTPPortAggregator, nil))
}
