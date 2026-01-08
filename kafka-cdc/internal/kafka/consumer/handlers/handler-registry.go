package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	repo "github.com/k-code-yt/go-api-practice/kafka-cdc/internal/repos"
)

type Handler func(ctx context.Context, msg []byte, metadata *kafka.TopicPartition) error

type Registry struct {
	handlers map[repo.EventType]Handler
	mu       *sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[repo.EventType]Handler),
		mu:       new(sync.RWMutex),
	}
}

func (r *Registry) AddHandler(h Handler, event repo.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[event] = h
}

func (r *Registry) GetHandler(event repo.EventType) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[event]
	if !ok {
		msg := fmt.Sprintf("handler is missing for %s event type", event)
		return nil, fmt.Errorf(msg)
	}
	return h, nil
}
