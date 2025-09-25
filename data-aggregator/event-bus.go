package main

import "go-logistic-api/shared"

type AggregatorEventBus interface {
	Subscribe(topic string, handler func(shared.SensorData))
	Publish(topic string, data shared.SensorData)
}

type InMemoryAggregatorEventBus struct {
	dataCH chan shared.SensorData
}

type EventBugConfig struct {
	eventBusType shared.EventBusType
}

func EventBusFactory(c EventBugConfig) AggregatorEventBus {
	if c.eventBusType == shared.EventBusType_InMemory {
		return NewInMemoryAggregatorEventBus()
	}
	return nil
}

func NewInMemoryAggregatorEventBus() *InMemoryAggregatorEventBus {
	return &InMemoryAggregatorEventBus{
		dataCH: make(chan shared.SensorData, 128),
	}
}

func (eb *InMemoryAggregatorEventBus) Publish(topic string, data shared.SensorData) {
	eb.dataCH <- data
}

func (eb *InMemoryAggregatorEventBus) Subscribe(topic string, handler func(shared.SensorData)) {
	for v := range eb.dataCH {
		handler(v)
	}
}
