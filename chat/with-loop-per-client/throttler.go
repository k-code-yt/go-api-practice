package withloopperclient

import (
	"fmt"
	"math/rand/v2"
	"time"
)

const (
	ThrottlerMessagesPerSecond = 100
)

type Throttler struct {
	inputCH  chan *ReqMsg
	outputCH chan *ReqMsg
	exit     chan struct{}
	rate     time.Duration
}

func NewThrottler(msgPerSec int) *Throttler {
	rate := time.Second / time.Duration(msgPerSec)

	t := &Throttler{
		inputCH:  make(chan *ReqMsg, 1),
		outputCH: make(chan *ReqMsg, 64),
		exit:     make(chan struct{}),
		rate:     rate,
	}

	go t.leak()

	return t
}

func (t *Throttler) leak() {
	ticker := time.NewTicker(t.rate)
	defer ticker.Stop()
	defer close(t.inputCH)
	defer close(t.outputCH)

	for {
		select {
		case <-t.exit:
			fmt.Println("exiting leak goroutine!")
			return
		case <-ticker.C:
			select {
			case msg := <-t.inputCH:
				t.outputCH <- msg
			default:
				if rand.IntN(10) < 1 {
					fmt.Println("no msg in the input -> skipping")
				}
			}
		}
	}
}

func (t *Throttler) Allow(msg *ReqMsg) bool {
	select {
	case t.inputCH <- msg:
		return true
	default:
		return false
	}
}

func (c *Client) acceptThrottledMsgLoop(handler func(msg *ReqMsg)) {
	for {
		select {
		case <-c.done:
			fmt.Printf("leaving acceptThrottledMsgLoop on CLIENT_DONE, cID = %s\n", c.ID)
			return
		case msg, ok := <-c.throttler.outputCH:
			if !ok {
				fmt.Printf("leaving acceptThrottledMsgLoop on THROTTLE_EXIT, cID = %s\n", c.ID)
				return
			}
			handler(msg)
		}
	}
}
