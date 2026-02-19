package main

import (
	"testing"
)

var res float64

func BenchmarkMathPow(b *testing.B) {
	boids := make([]*Boid, boidsCount)
	for id := range boidsCount {
		b := NewBoid(id, nil)
		boids[id] = b
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sum float64 = 0.0
	for i := 0; i < b.N; i++ {
		idx := i % boidsCount
		b := boids[idx]
		sum += b.position.x*b.position.x + b.position.y*b.position.y
	}
	res = sum
}
