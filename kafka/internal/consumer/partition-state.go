package consumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

type PartitionState struct {
	ID           int32
	state        map[kafka.Offset]MsgState
	maxReceived  *kafka.TopicPartition
	mu           *sync.RWMutex
	lastCommited kafka.Offset
	ctx          context.Context
	Cancel       context.CancelFunc
	ExitCH       chan struct{}
}

func NewPartitionState(maxReceived *kafka.TopicPartition) *PartitionState {
	ctx, Cancel := context.WithCancel(context.Background())
	initialLastCommited := maxReceived.Offset - 1
	if maxReceived.Offset == kafka.OffsetBeginning || maxReceived.Offset < 0 {
		initialLastCommited = -1
	}
	return &PartitionState{
		ID:           maxReceived.Partition,
		state:        map[kafka.Offset]MsgState{},
		maxReceived:  maxReceived,
		lastCommited: initialLastCommited,
		mu:           &sync.RWMutex{},
		ctx:          ctx,
		Cancel:       Cancel,
		ExitCH:       make(chan struct{}),
	}
}

func (ps *PartitionState) commitOffsetLoop(commitDur time.Duration, c *KafkaConsumer) {
	ticker := time.NewTicker(commitDur)
	defer func() {
		close(ps.ExitCH)
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

			latestToCommit, err := ps.FindLatestToCommit()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = c.consumer.CommitOffsets([]kafka.TopicPartition{*latestToCommit})
			if err != nil {
				fmt.Printf("err commiting offset = %d, prtn = %d, err = %v\n", latestToCommit.Offset, ps.maxReceived.Partition, err)
				continue
			}

			ps.mu.Lock()
			ps.lastCommited = latestToCommit.Offset
			if ps.lastCommited > ps.maxReceived.Offset {
				ps.maxReceived.Offset = latestToCommit.Offset
			}
			logrus.WithFields(
				logrus.Fields{
					"COMMITED_OFFSET": latestToCommit.Offset,
					"MAX_OFFSET":      ps.maxReceived.Offset,
					"PRTN":            ps.maxReceived.Partition,
					"STATE":           ps.state,
				},
			).Warn("Commited on CRON")
			ps.mu.Unlock()

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionState) FindLatestToCommit() (*kafka.TopicPartition, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// fmt.Printf("PRTN = %d, STATE = %+v\n", ps.ID, ps.state)

	if ps.maxReceived == nil {
		return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.maxReceived
	if ps.lastCommited > ps.maxReceived.Offset {
		// panic("❌last commit above maxReceived❌")
	}
	if ps.lastCommited == ps.maxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == maxReceived in prtn %d -> skipping\n", ps.lastCommited, ps.maxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}
	for offset := ps.lastCommited; offset <= ps.maxReceived.Offset; offset++ {
		msgState, exists := ps.state[offset]
		if !exists {
			// fmt.Printf("does not exit, off = %d, state = %v\n", offset, ps.state)
			continue
		}
		if msgState != MsgState_Pending {
			delete(ps.state, offset)
			// logrus.WithFields(logrus.Fields{
			// 	"OFFSET": offset,
			// 	"PRTN":   ps.ID,
			// }).Info("Removed offset")
			if len(ps.state) == 0 {
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
