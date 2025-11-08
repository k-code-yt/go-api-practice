package ratelimitter

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type RateLimiter struct {
	refillRate float64
	lastRefill time.Time
	tokens     float64
	maxTokens  float64
	mu         *sync.RWMutex
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	r := &RateLimiter{
		refillRate: refillRate,
		mu:         new(sync.RWMutex),
		maxTokens:  maxTokens,
		tokens:     maxTokens,
		lastRefill: time.Now(),
	}

	return r
}

func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()
	if r.tokens >= 1 {
		r.tokens--
		return true
	}

	return false
}

func (r *RateLimiter) refill() {
	ellapsed := time.Since(r.lastRefill).Seconds()
	toRefill := ellapsed * r.refillRate

	r.tokens = math.Min(r.maxTokens, r.tokens+toRefill)
	r.lastRefill = time.Now()
}

type PerClientLimiter struct {
	limiters      map[string]*RateLimiter
	mu            *sync.RWMutex
	cleanUpTicker *time.Ticker
	maxInactive   time.Duration
	ctx           context.Context
}

func NewPerClientLimiter(ctx context.Context) *PerClientLimiter {
	rl := &PerClientLimiter{
		limiters:      make(map[string]*RateLimiter),
		mu:            new(sync.RWMutex),
		cleanUpTicker: time.NewTicker(time.Second * 1),
		maxInactive:   time.Second * 2,
		ctx:           ctx,
	}

	go rl.cleanUp()
	return rl
}

func (c *PerClientLimiter) Allow(cID string) bool {
	c.mu.RLock()
	rl, ok := c.limiters[cID]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		rl = NewRateLimiter(MaxTokens, RefillRatePerSecond)
		c.limiters[cID] = rl
		c.mu.Unlock()
	}

	return rl.Allow()
}

func (c *PerClientLimiter) cleanUp() {
	defer c.cleanUpTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("===EXIT-CLEAN-UP-GOROUTINE===")
			return
		case <-c.cleanUpTicker.C:
			fmt.Println("===CLEAN-UP-START===")
			c.mu.Lock()
			for id, rl := range c.limiters {
				sinceT := time.Since(rl.lastRefill)
				if sinceT >= c.maxInactive {
					delete(c.limiters, id)
					fmt.Printf("removed cID from limiters due to inactivity %s\n", id)
				}
			}
			c.mu.Unlock()
		}
	}
}
