package main

import (
	"encoding/json"
	"go-logistic-api/shared"
	"log"
	"net/http"
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
	aggrService := NewAggregatorService(aggrStore, eventBus)

	eventBus.CreateTopic("invoice-calculator")
	eventBus.CreateTopic("distance-calculator")
	go msgBroker.consumer.ReadMessageLoop()
	go eventBus.Subscribe("invoice-calculator", aggrService.AcceptSensorData)
	go eventBus.Subscribe("distance-calculator", aggrService.AggregateDistance)

	http.HandleFunc("/get-distance", handleGetDistance(aggrService))
	log.Fatal(http.ListenAndServe(shared.HTTPPortAggregator, nil))
}
