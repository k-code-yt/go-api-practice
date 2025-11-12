package ratelimitter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	HTTPPort                    = ":6551"
	MaxTokens           float64 = 1000000
	RefillRatePerSecond float64 = 1000000
)

type HTTPServer struct {
	rl     *PerClientLimiter
	mu     *sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewHTTPServer(ctx context.Context, cancel context.CancelFunc, rl *PerClientLimiter) *HTTPServer {
	return &HTTPServer{
		rl:     rl,
		mu:     new(sync.RWMutex),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *HTTPServer) handler(w http.ResponseWriter, r *http.Request) {
	cID := r.Header.Get("x-client-id")
	if cID == "" {
		log.Fatal("missing clientID")
	}

	isAllowed := s.rl.Allow(cID)
	if !isAllowed {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HTTPServer) RunHTTPServer() {

	http.HandleFunc("/rl", s.handler)

	fmt.Println("listening on port = ", HTTPPort)
	log.Fatal(http.ListenAndServe(HTTPPort, nil))
}

func CreateServer() *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	perClientRL := NewPerClientLimiter(ctx)
	s := NewHTTPServer(ctx, cancel, perClientRL)
	return s
}
