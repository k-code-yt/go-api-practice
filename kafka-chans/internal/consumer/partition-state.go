package consumer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-chans/internal/shared"
	"github.com/sirupsen/logrus"
)

type UpdateStateMsg struct {
	offset kafka.Offset
	value  shared.MsgState
}

func NewUpdateStateMsg(offset kafka.Offset, value shared.MsgState) *UpdateStateMsg {
	return &UpdateStateMsg{
		offset: offset,
		value:  value,
	}
}

func (s *UpdateStateMsg) SetOffset(offset kafka.Offset) {
	s.offset = offset
}

func (s *UpdateStateMsg) SetStateValue(value shared.MsgState) {
	s.value = value
}

type PartitionState struct {
	ID           int32
	state        map[kafka.Offset]shared.MsgState
	stateSize    *atomic.Int64
	MaxReceived  *kafka.TopicPartition
	lastCommited kafka.Offset

	DeleteFromStateCH        chan kafka.Offset
	UpdateStateCH            chan *UpdateStateMsg
	GetStateSizeCH           chan chan int
	FindLatestToCommitReqCH  chan struct{}
	FindLatestToCommitRespCH chan kafka.Offset

	ReadOffsetReqCH  chan kafka.Offset
	ReadOffsetRespCH chan shared.MsgState

	ctx    context.Context
	Cancel context.CancelFunc
	exitCH chan struct{}

	MsgPool *sync.Pool
}

func NewPartitionState(MaxReceived *kafka.TopicPartition) *PartitionState {
	ctx, Cancel := context.WithCancel(context.Background())
	initialLastCommited := MaxReceived.Offset - 1
	if MaxReceived.Offset == kafka.OffsetBeginning || MaxReceived.Offset < 0 {
		initialLastCommited = -1
	}
	ps := &PartitionState{
		ID:           MaxReceived.Partition,
		state:        map[kafka.Offset]shared.MsgState{},
		stateSize:    new(atomic.Int64),
		MaxReceived:  MaxReceived,
		lastCommited: initialLastCommited,
		ctx:          ctx,
		Cancel:       Cancel,
		exitCH:       make(chan struct{}),

		DeleteFromStateCH: make(chan kafka.Offset, 128),
		UpdateStateCH:     make(chan *UpdateStateMsg, 128),
		GetStateSizeCH:    make(chan chan int, 16),

		// TODO -> remove - only for testing
		FindLatestToCommitReqCH:  make(chan struct{}, 1028),
		FindLatestToCommitRespCH: make(chan kafka.Offset, 1028),
	}

	go ps.acceptMsgLoop()
	return ps
}

func (ps *PartitionState) GetState() map[kafka.Offset]shared.MsgState {
	return ps.state
}

func (ps *PartitionState) CommitOffsetLoop(commitDur time.Duration, c *KafkaConsumer) {
	ticker := time.NewTicker(commitDur)
	defer func() {
		close(ps.exitCH)
		ticker.Stop()
		logrus.WithFields(
			logrus.Fields{
				"PRTN": ps.ID,
			},
		).Info("EXIT commitOffsetLoop✅")
	}()
	for {
		fmt.Println("------RUNNING-COMMIT-LOOP------")
		select {
		case <-ticker.C:
			select {
			case <-ps.ctx.Done():
				return
			default:
			}

			// stateCopy := maps.Clone(ps.state)
			latestToCommit, err := ps.FindLatestToCommit()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = c.consumer.CommitOffsets([]kafka.TopicPartition{*latestToCommit})
			if err != nil {
				fmt.Printf("err commiting offset = %d, prtn = %d, err = %v\n", latestToCommit.Offset, ps.MaxReceived.Partition, err)
				continue
			}

			ps.lastCommited = latestToCommit.Offset
			if ps.lastCommited > ps.MaxReceived.Offset {
				ps.MaxReceived.Offset = latestToCommit.Offset
			}
			logrus.WithFields(
				logrus.Fields{
					"COMMITED_OFFSET": latestToCommit.Offset,
					"MAX_OFFSET":      ps.MaxReceived.Offset,
					"PRTN":            ps.MaxReceived.Partition,
					"STATE":           ps.state,
				},
			).Warn("Commited on CRON")

		case <-ps.ctx.Done():
			return
		}
	}
}

// func (ps *PartitionState) FindLatestToCommit(stateCopy map[kafka.Offset]shared.MsgState) (*kafka.TopicPartition, error) {
func (ps *PartitionState) FindLatestToCommit() (*kafka.TopicPartition, error) {
	// defer func() {
	// 	stateCopy = nil
	// }()
	// fmt.Printf("PRTN = %d, STATE = %+v\n", ps.ID, ps.state)

	if ps.MaxReceived == nil {
		// return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.MaxReceived
	if ps.lastCommited > ps.MaxReceived.Offset {
		// fmt.Println("❌last commit above maxReceived❌")
		// fmt.Printf("last = %d, max = %d\n", ps.lastCommited, ps.MaxReceived.Offset)
		// panic("❌last commit above maxReceived❌")
	}
	if ps.lastCommited == ps.MaxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == MaxReceived in prtn %d -> skipping\n", ps.lastCommited, ps.MaxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}

	for offset := ps.lastCommited; offset <= ps.MaxReceived.Offset+1; offset++ {
		if ps.getStateSize() == 0 {
			latestToCommit.Offset = offset + 1
			break
		}

		msgState, exists := ps.state[offset]
		if !exists {
			// fmt.Printf("does not exit, off = %d, state = %v\n", offset, ps.state)
			continue
		}
		if msgState != shared.MsgState_Pending {
			delete(ps.state, offset)
			continue
		}
		latestToCommit.Offset = offset
		break
	}
	return &latestToCommit, nil
}

func (ps *PartitionState) acceptMsgLoop() {
	defer func() {
		fmt.Println("EXIT acceptMsgLoop")
	}()
	for {
		select {
		case <-ps.ctx.Done():
			return
		case offset := <-ps.DeleteFromStateCH:
			delete(ps.state, offset)
			ps.stateSize.Add(-1)
			logrus.WithFields(logrus.Fields{
				"OFFSET": offset,
				"PRTN":   ps.ID,
			}).Info("Removed offset")
		case msg := <-ps.UpdateStateCH:
			ps.state[msg.offset] = msg.value
			ps.stateSize.Add(1)
			ps.MsgPool.Put(msg)
		case <-ps.FindLatestToCommitReqCH:
			tp, _ := ps.FindLatestToCommit()
			ps.FindLatestToCommitRespCH <- tp.Offset
		case offset := <-ps.ReadOffsetReqCH:
			v, ok := ps.state[offset]
			if !ok {
				ps.ReadOffsetRespCH <- -1
			}
			ps.ReadOffsetRespCH <- v
		}
	}
}

func (ps *PartitionState) getStateSize() int {
	return int(ps.stateSize.Load())
}

// TODO  -> remove
// only for bench-testing
func NewTestPartitionState(MaxReceived *kafka.TopicPartition) *PartitionState {
	ctx, Cancel := context.WithCancel(context.Background())

	initialLastCommited := MaxReceived.Offset - 1
	if MaxReceived.Offset == kafka.OffsetBeginning || MaxReceived.Offset < 0 {
		initialLastCommited = -1
	}
	ps := &PartitionState{
		ID:           MaxReceived.Partition,
		state:        map[kafka.Offset]shared.MsgState{},
		stateSize:    new(atomic.Int64),
		MaxReceived:  MaxReceived,
		lastCommited: initialLastCommited,
		exitCH:       make(chan struct{}),
		ctx:          ctx,
		Cancel:       Cancel,

		DeleteFromStateCH: make(chan kafka.Offset, 128),
		UpdateStateCH:     make(chan *UpdateStateMsg, 128),
		GetStateSizeCH:    make(chan chan int, 16),

		// TODO -> remove - only for testing
		FindLatestToCommitReqCH:  make(chan struct{}, 1),
		FindLatestToCommitRespCH: make(chan kafka.Offset, 1),
		ReadOffsetReqCH:          make(chan kafka.Offset, 1),
		ReadOffsetRespCH:         make(chan shared.MsgState, 1),
	}

	var msgPool = sync.Pool{
		New: func() interface{} {
			return &UpdateStateMsg{}
		},
	}

	ps.MsgPool = &msgPool

	go ps.acceptMsgLoop()
	return ps
}
