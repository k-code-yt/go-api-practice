package main

import (
	"context"
	"sync"

	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
)

// TODO -> rework to be generic with any type
type AggregatorEventBus interface {
	Subscribe(topic shared.DomainEvent, handler func(*shared.SensorData) error)
	Publish(topic shared.DomainEvent, data *shared.SensorData)
}

type EventBusSub struct {
	dataCH  chan *shared.SensorData
	handler func(*shared.SensorData) error
	topic   shared.DomainEvent
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewEventBusSub(
	topic shared.DomainEvent,
	handler func(*shared.SensorData) error,
) *EventBusSub {

	ctx, cancel := context.WithCancel(context.Background())
	dataCH := make(chan *shared.SensorData)
	return &EventBusSub{
		dataCH:  dataCH,
		topic:   topic,
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
	}
}

type InMemoryAggregatorEventBus struct {
	subs   []*EventBusSub
	subMap map[shared.DomainEvent]*EventBusSub
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
		subMap: make(map[shared.DomainEvent]*EventBusSub),
		mu:     new(sync.RWMutex),
	}
}

func (eb *InMemoryAggregatorEventBus) Publish(topic shared.DomainEvent, data *shared.SensorData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	sub, ok := eb.subMap[topic]
	if !ok {
		logrus.Error("Topic does not exist, requires at least one subscriber")
		return
	}

	sub.dataCH <- data
}

func (eb *InMemoryAggregatorEventBus) PublishAll(data *shared.SensorData) {
	for _, sub := range eb.subs {
		sub.dataCH <- data
	}
}

func (eb *InMemoryAggregatorEventBus) Subscribe(topic shared.DomainEvent, handler func(*shared.SensorData) error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub, ok := eb.subMap[topic]
	if ok {
		logrus.Infof("Topic %s already registered\noverwriting handler to the new one", topic)
		sub.handler = handler
	}

	if !ok {
		sub = NewEventBusSub(topic, handler)
		eb.subMap[topic] = sub
		eb.subs = append(eb.subs, sub)
	}
	go eb.processEvent(sub)
}

func (eb *InMemoryAggregatorEventBus) processEvent(sub *EventBusSub) {
	defer close(sub.dataCH)

	for {
		select {
		case v := <-sub.dataCH:
			err := sub.handler(v)
			if err != nil {
				logrus.Errorf("error eventbus handler %v\n", err)
				continue
			}
		case <-sub.ctx.Done():
			logrus.Warnf("Exiting InternalEventBus Topic =  %s\n", sub.topic)
			return
		}
	}
}
