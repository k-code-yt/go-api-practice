package main

import (
	"sync"

	"github.com/k-code-yt/go-api-practice/shared"
)

// TODO -> rework to be generic with any type
type AggregatorEventBus interface {
	Subscribe(topic string, handler func(*shared.SensorData))
	Publish(data *shared.SensorData)
	CreateTopic(topic string)
}

type EventBusSub struct {
	dataCH  chan *shared.SensorData
	handler func(*shared.SensorData)
	topic   string
}

type InMemoryAggregatorEventBus struct {
	subs   []*EventBusSub
	subMap map[string]*EventBusSub
	mu     *sync.RWMutex
}

type EventBusConfig struct {
	eventBusType shared.EventBusType
}

func EventBusFactory(c EventBusConfig) AggregatorEventBus {
	if c.eventBusType == shared.EventBusType_InMemory {
		return NewInMemoryAggregatorEventBus()
	}
	return nil
}

func NewInMemoryAggregatorEventBus() *InMemoryAggregatorEventBus {
	var subs []*EventBusSub
	return &InMemoryAggregatorEventBus{
		subs:   subs,
		subMap: make(map[string]*EventBusSub),
		mu:     new(sync.RWMutex),
	}
}

func (eb *InMemoryAggregatorEventBus) Publish(data *shared.SensorData) {
	for _, sub := range eb.subs {
		sub.dataCH <- data
	}
}

func (eb *InMemoryAggregatorEventBus) Subscribe(topic string, handler func(*shared.SensorData)) {
	eb.registerHandler(topic, handler)

	for _, sub := range eb.subs {
		go func(sub *EventBusSub) {
			for v := range sub.dataCH {
				sub.handler(v)
			}
		}(sub)
	}
}

func (eb *InMemoryAggregatorEventBus) CreateTopic(topic string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	_, ok := eb.subMap[topic]
	if !ok {
		sub := &EventBusSub{
			topic:  topic,
			dataCH: make(chan *shared.SensorData, 64),
		}
		eb.subMap[topic] = sub
		eb.subs = append(eb.subs, sub)
	}
}

func (eb *InMemoryAggregatorEventBus) registerHandler(topic string, handler func(*shared.SensorData)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	sub, ok := eb.subMap[topic]
	if !ok {
		sub := &EventBusSub{
			topic:   topic,
			handler: handler,
			dataCH:  make(chan *shared.SensorData, 64),
		}
		eb.subMap[topic] = sub
	} else {
		sub.handler = handler
	}
}
