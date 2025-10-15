package main

import (
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
)

type InMemoryStore struct {
	distances []shared.Distance
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		distances: []shared.Distance{},
	}
}

func (s *InMemoryStore) Insert(d shared.Distance) (string, error) {
	s.distances = append(s.distances, d)
	logrus.Infof("stored distance = %v\n", s.distances[len(s.distances)-1])
	return d.ID, nil
}

func (s *InMemoryStore) GetDistance(id string) *shared.Distance {
	var d shared.Distance
	for _, v := range s.distances {
		if v.ID == id {
			d = v
			break
		}
	}

	return &d
}
