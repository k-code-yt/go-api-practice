package withloopperclient

import "sync"

type RingBuffer struct {
	buffer []*RespMsg
	size   int
	head   int
	count  int
	mu     *sync.RWMutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		size:   size,
		mu:     new(sync.RWMutex),
		buffer: make([]*RespMsg, size),
	}
}

func (rb *RingBuffer) Push(msg *RespMsg) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = msg
	rb.head = (rb.head + 1) % rb.size

	if rb.count < rb.size {
		rb.count++
	}
}

func (rb *RingBuffer) GetFromOffset(offset int) []*RespMsg {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return []*RespMsg{}
	}

	oldestIdx := (rb.head - rb.count + rb.size) % rb.size

	startPos := -1

	for i := 0; i < rb.count; i++ {
		idx := (oldestIdx + 1) % rb.size
		if rb.buffer[idx].Offset > offset {
			startPos = i
		}
	}

	if startPos == -1 {
		return []*RespMsg{}
	}

	numMsg := rb.count - startPos
	result := make([]*RespMsg, numMsg)
	for i := range numMsg {
		idx := (oldestIdx + startPos + i) % rb.size
		result[i] = rb.buffer[idx]
	}

	return result
}

func (rb *RingBuffer) GetAll() []*RespMsg {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return []*RespMsg{}
	}

	oldestIdx := (rb.head - rb.count + rb.size) % rb.size
	result := make([]*RespMsg, rb.count)
	for i := range rb.count {
		idx := (oldestIdx + i) % rb.size
		result[i] = rb.buffer[idx]
	}

	return result
}
