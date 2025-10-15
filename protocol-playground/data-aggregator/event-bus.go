package main

import (
	"context"
	"sync"

	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
)

type EventBusConfig struct {
	eventBusType shared.EventBusType
}

type EventHandler[T any] func(ctx context.Context, event T) error

type AggregatorEventBus[T any] interface {
	Subscribe(topic shared.DomainEvent, handler EventHandler[T])
	Publish(topic shared.DomainEvent, data T)
	Close()
	IsActive() bool
}

type EventBusSub[T any] struct {
	dataCH  chan T
	handler EventHandler[T]
	topic   shared.DomainEvent
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewEventBusSub[T any](
	topic shared.DomainEvent,
	handler EventHandler[T],
) *EventBusSub[T] {

	ctx, cancel := context.WithCancel(context.Background())
	dataCH := make(chan T, 32)
	return &EventBusSub[T]{
		dataCH:  dataCH,
		topic:   topic,
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
	}
}

type InMemoryAggregatorEventBus[T any] struct {
	subs     []*EventBusSub[T]
	subMap   map[shared.DomainEvent]*EventBusSub[T]
	mu       *sync.RWMutex
	isActive bool
}

func NewInMemoryAggregatorEventBus[T any]() *InMemoryAggregatorEventBus[T] {
	var subs []*EventBusSub[T]
	return &InMemoryAggregatorEventBus[T]{
		subs:     subs,
		subMap:   make(map[shared.DomainEvent]*EventBusSub[T]),
		mu:       new(sync.RWMutex),
		isActive: true,
	}
}

func EventBusFactory[T any](c EventBusConfig) AggregatorEventBus[T] {
	if c.eventBusType == shared.EventBusType_InMemory {
		return NewInMemoryAggregatorEventBus[T]()
	}
	return nil
}

func (eb *InMemoryAggregatorEventBus[T]) Publish(topic shared.DomainEvent, data T) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	sub, ok := eb.subMap[topic]
	if !ok {
		logrus.Error("Topic does not exist, requires at least one subscriber")
		return
	}

	sub.dataCH <- data
}

func (eb *InMemoryAggregatorEventBus[T]) Subscribe(topic shared.DomainEvent, handler EventHandler[T]) {
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

func (eb *InMemoryAggregatorEventBus[T]) processEvent(sub *EventBusSub[T]) {
	defer func() {
		logrus.Info("No more events to process -> closing eventBus Chan")
		eb.isActive = false
		close(sub.dataCH)
	}()

	for {
		select {
		case v := <-sub.dataCH:
			err := sub.handler(sub.ctx, v)
			if err != nil {
				logrus.Errorf("error eventbus handler %v\n", err)
				continue
			}
		case <-sub.ctx.Done():
			logrus.WithField("event", sub.topic).Warn("Exiting InternalEventBus on SIGTERM")
			return
		}
	}

}

func (eb *InMemoryAggregatorEventBus[T]) Close() {
	for _, sub := range eb.subs {
		logrus.WithFields(logrus.Fields{
			"event": sub.topic,
		}).Warn("Close chan on SIGTERM")
		sub.cancel()

	}
}

func (eb *InMemoryAggregatorEventBus[T]) IsActive() bool {
	return eb.isActive
}
