package withloopperclient

import (
	"fmt"
	"time"
)

type BPStrategy int

const (
	BPStrategy_Drop BPStrategy = iota
	BPStrategy_Block
	BPStrategy_SlowDown
)

const (
	DefaultBackPressureStrategy = BPStrategy_Drop
	MaxBackPressureQueueSize    = 5
)

func (c *Client) handleBackpressure(msg *RespMsg, droppedCH chan<- struct{}) {
	qs := int(c.queueSize.Load())

	switch c.bpStrategy {
	case BPStrategy_SlowDown:
		select {
		case c.msgCH <- msg:
			c.queueSize.Add(1)
		case <-time.After(100 * time.Millisecond):
			c.droppedMsgCount.Add(1)
			droppedCH <- struct{}{}
			fmt.Printf("dropped msg - SLOW_CLIENT, cID = %s\n", c.ID)
			return
		}

	case BPStrategy_Drop:
		if qs >= MaxBackPressureQueueSize {
			fmt.Printf("dropped msg - MAX_QUEUE_SIZE_WAS_REACHED, cID = %s\n", c.ID)
			droppedCH <- struct{}{}
			c.droppedMsgCount.Add(1)
			return
		}
		c.queueSize.Add(1)
		c.msgCH <- msg

	case BPStrategy_Block:
		c.queueSize.Add(1)
		c.msgCH <- msg
	}
}
