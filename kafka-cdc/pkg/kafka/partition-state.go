package pkgkafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

type CommitFunc func([]kafka.TopicPartition) ([]kafka.TopicPartition, error)

type PartitionState struct {
	ID           int32
	State        map[kafka.Offset]MsgState
	MaxReceived  *kafka.TopicPartition
	Mu           *sync.RWMutex
	LastCommited kafka.Offset
	commitFunc   CommitFunc

	ctx    context.Context
	Cancel context.CancelFunc
	ExitCH chan struct{}
}

func NewPartitionState(MaxReceived *kafka.TopicPartition, commitFunc CommitFunc) *PartitionState {
	ctx, Cancel := context.WithCancel(context.Background())
	initialLastCommited := MaxReceived.Offset - 1
	if MaxReceived.Offset == kafka.OffsetBeginning || MaxReceived.Offset < 0 {
		initialLastCommited = -1
	}
	return &PartitionState{
		ID:           MaxReceived.Partition,
		Mu:           &sync.RWMutex{},
		State:        map[kafka.Offset]MsgState{},
		MaxReceived:  MaxReceived,
		LastCommited: initialLastCommited,
		commitFunc:   commitFunc,

		ctx:    ctx,
		Cancel: Cancel,
		ExitCH: make(chan struct{}),
	}
}

func (ps *PartitionState) commitOffsetLoop(commitDur time.Duration) {
	ticker := time.NewTicker(commitDur)
	defer func() {
		close(ps.ExitCH)
		ticker.Stop()
		// logrus.WithFields(
		// 	logrus.Fields{
		// 		"PRTN": ps.ID,
		// 	},
		// ).Info("EXIT commitOffsetLoop✅")
	}()
	for {
		// fmt.Println("------RUNNING-COMMIT-LOOP------")
		select {
		case <-ticker.C:
			select {
			case <-ps.ctx.Done():
				return
			default:
			}

			latestToCommit, err := ps.FindLatestToCommit()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = ps.commitFunc([]kafka.TopicPartition{*latestToCommit})
			if err != nil {
				ps.Mu.RLock()
				fmt.Printf("err commiting offset = %d, prtn = %d, err = %v\n", latestToCommit.Offset, ps.MaxReceived.Partition, err)
				ps.Mu.RUnlock()
				continue
			}

			ps.Mu.Lock()
			ps.LastCommited = latestToCommit.Offset
			if ps.LastCommited > ps.MaxReceived.Offset {
				ps.MaxReceived.Offset = latestToCommit.Offset
			}
			logrus.WithFields(
				logrus.Fields{
					"COMMITED_OFFSET": latestToCommit.Offset,
					"MAX_OFFSET":      ps.MaxReceived.Offset,
					"PRTN":            ps.MaxReceived.Partition,
					"STATE":           ps.State,
				},
			).Warn("Commited on CRON")
			ps.Mu.Unlock()

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionState) FindLatestToCommit() (*kafka.TopicPartition, error) {
	ps.Mu.Lock()
	defer ps.Mu.Unlock()
	// fmt.Printf("PRTN = %d, STATE = %+v\n", ps.ID, ps.State)

	if ps.MaxReceived == nil {
		return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.MaxReceived
	if ps.LastCommited == ps.MaxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == MaxReceived in prtn %d -> skipping\n", ps.LastCommited, ps.MaxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}
	for offset := ps.LastCommited; offset <= ps.MaxReceived.Offset; offset++ {
		msgState, exists := ps.State[offset]
		if !exists {
			// fmt.Printf("does not exit, off = %d, State = %v\n", offset, ps.State)
			continue
		}
		if msgState != MsgState_Pending {
			delete(ps.State, offset)
			logrus.WithFields(logrus.Fields{
				"OFFSET": offset,
				"PRTN":   ps.ID,
			}).Info("Removed offset")
			if len(ps.State) == 0 {
				latestToCommit.Offset = offset + 1
				break
			}
			continue
		}
		latestToCommit.Offset = offset
		break
	}
	return &latestToCommit, nil
}

func (ps *PartitionState) ReadOffset(offset kafka.Offset) (MsgState, bool) {
	ps.Mu.RLock()
	defer ps.Mu.RUnlock()

	state, exists := ps.State[offset]
	return state, exists
}
