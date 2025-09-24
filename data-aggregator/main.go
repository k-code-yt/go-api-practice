package main

import (
	"go-logistic-api/shared"
)

func main() {
	msgBroker, err := NewMsgBroker(&MsgBrokerConfig{
		brokerType: shared.MsgBrokerType_Kafka,
	})
	if err != nil {
		panic(err)
	}

	aggrService := NewAggregatorService()

	go msgBroker.consumer.ReadMessageLoop(aggrService.dataCH)

	aggrService.ReadMessageLoop()
}
