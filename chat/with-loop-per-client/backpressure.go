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

func (c *Server) backpressureSendMsg(msg *ReqMsg, cls map[string]*Client) {
	resp := NewRespMsg(msg)
	for _, c := range cls {
		go c.handleBackpressure(resp)
	}

	m := msg.RoomID
	if msg.RoomID == "" {
		m = "BROADCAST"
	}
	fmt.Printf("msg was sent to rID= %s | by cID= %s | num_clients=%d\n", m, msg.Client.ID, len(cls))
}

func (c *Client) handleBackpressure(msg *RespMsg) {
	qs := int(c.queueSize.Load())

	switch c.bpStrategy {
	case BPStrategy_SlowDown:
		select {
		case c.msgCH <- msg:
			c.queueSize.Add(1)
		case <-time.After(100 * time.Millisecond):
			c.droppedMsgCount.Add(1)
			fmt.Printf("dropped msg - SLOW_CLIENT, cID = %s\n", c.ID)
			return
		}

	case BPStrategy_Drop:
		if qs >= MaxBackPressureQueueSize {
			fmt.Printf("dropped msg - MAX_QUEUE_SIZE_WAS_REACHED, cID = %s\n", c.ID)
			return
		}
		c.msgCH <- msg
		c.queueSize.Add(1)

	case BPStrategy_Block:
		c.msgCH <- msg
		c.queueSize.Add(1)
	}
}

func (c *Client) getBackpressureStats() map[string]interface{} {
	return map[string]interface{}{
		"queueSize":       c.queueSize.Load(),
		"droppedMsgCount": c.droppedMsgCount.Load(),
	}
}
