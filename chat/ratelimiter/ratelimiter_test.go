package ratelimitter

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestClient struct {
	cID      int
	rejCount *atomic.Int64
	reqCount int
}

func NewTestClient(cID int, reqCount int) *TestClient {
	return &TestClient{
		cID:      cID,
		rejCount: new(atomic.Int64),
		reqCount: reqCount,
	}
}

func (c *TestClient) sendReqs(wg *sync.WaitGroup) {
	defer wg.Done()
	localWG := sync.WaitGroup{}
	localWG.Add(c.reqCount)
	for range c.reqCount {
		go func() {
			defer localWG.Done()
			req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost%s/rl", HTTPPort), nil)
			if err != nil {
				log.Fatal(err)
			}
			req.Header.Set("x-client-id", strconv.Itoa(c.cID))

			cl := &http.Client{}

			r, err := cl.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer r.Body.Close()

			status := r.StatusCode

			if status == http.StatusTooManyRequests {
				c.rejCount.Add(1)
			}
			fmt.Printf("completed cID = %d, status =%d\n", c.cID, status)
		}()
	}
	localWG.Wait()
}

func TestRequestRateLimitter(t *testing.T) {
	wg := new(sync.WaitGroup)
	reqCount := 10
	clientCount := 5
	clients := []*TestClient{}

	wg.Add(clientCount)

	s := CreateServer()
	go s.RunHTTPServer()
	time.Sleep(time.Second)

	for idx := range clientCount {
		reqCount := rand.IntN(reqCount) + 5
		c := NewTestClient(idx, reqCount)
		clients = append(clients, c)
		go c.sendReqs(wg)
	}

	wg.Wait()
	for _, c := range clients {
		rc := int(c.rejCount.Load())
		fmt.Printf("rejected count = %d, reqCount = %d, cID = %d\n", rc, c.reqCount, c.cID)
		assert.Greater(t, rc, 0, "more then 0 rejected count")
		assert.Greater(t, c.reqCount, rc, "total request are greater then rejected")
	}
	time.Sleep(2 * time.Second)

	wg.Add(1)
	go clients[0].sendReqs(wg)
	wg.Wait()

	time.Sleep(2 * time.Second)

	s.rl.mu.Lock()
	limitersCount := len(s.rl.limiters)
	s.rl.mu.Unlock()
	fmt.Printf("count of limiters %d\n", limitersCount)
	assert.Equal(t, 1, limitersCount, "removed all limiters except one")

	s.cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("====TEST DONE====")
}

func BenchmarkRateLimiter_SingleClient(b *testing.B) {
	rl := NewRateLimiter(1000000, 1000000) // Large values to avoid limiting

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow()
		}
	})
}

func BenchmarkPerClientLimiter_SingleClient(b *testing.B) {
	ctx := b.Context()

	pcl := NewPerClientLimiter(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pcl.Allow("client1")
		}
	})
}

func BenchmarkPerClientLimiter_ManyClients(b *testing.B) {
	ctx := b.Context()

	pcl := NewPerClientLimiter(ctx)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		clientID := fmt.Sprintf("client-%d", rand.IntN(1000))
		for pb.Next() {
			pcl.Allow(clientID)
		}
	})
}

func BenchmarkPerClientLimiter_Creation(b *testing.B) {
	ctx := b.Context()

	pcl := NewPerClientLimiter(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client-%d", i) // ✅ Always new client
		pcl.Allow(clientID)
	}
}
