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
	cancel       context.CancelFunc
	exitCH       chan struct{}
}

func NewPartitionState(maxReceived *kafka.TopicPartition) *PartitionState {
	ctx, cancel := context.WithCancel(context.Background())
	return &PartitionState{
		ID:           maxReceived.Partition,
		state:        map[kafka.Offset]MsgState{},
		maxReceived:  maxReceived,
		lastCommited: maxReceived.Offset,
		mu:           &sync.RWMutex{},
		ctx:          ctx,
		cancel:       cancel,
		exitCH:       make(chan struct{}),
	}

}

func (ps *PartitionState) commitOffsetLoop(commitDur time.Duration, c *KafkaConsumer) {
	ticker := time.NewTicker(commitDur)
	defer func() {
		ticker.Stop()
		close(ps.exitCH)
		logrus.WithFields(
			logrus.Fields{
				"PRTN": ps.ID,
			},
		).Info("EXIT commitOffsetLoop")
	}()
	for {
		select {
		case <-ticker.C:
			latestToCommit, err := ps.findLatestToCommit()
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
			ps.lastCommited = latestToCommit.Offset - 1
			for offset, v := range ps.state {
				logrus.WithFields(
					logrus.Fields{
						"OFFSET":    offset,
						"PRTN":      ps.maxReceived.Partition,
						"STATE_VAL": v,
					},
				).Infof("STATE_AFTER_COMMIT")
			}
			ps.mu.Unlock()

			logrus.WithFields(
				logrus.Fields{
					"OFFSET": latestToCommit.Offset - 1,
					"PRTN":   ps.maxReceived.Partition,
				},
			).Warn("Commited on CRON")

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionState) findLatestToCommit() (*kafka.TopicPartition, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.maxReceived == nil {
		return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.maxReceived
	if ps.lastCommited > ps.maxReceived.Offset {
		panic("last commit above maxReceived")
	}

	if ps.lastCommited == ps.maxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == maxReceived in prtn %d -> skipping\n", ps.lastCommited, ps.maxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}

	for offset := ps.lastCommited; offset <= ps.maxReceived.Offset; offset++ {
		msgState, exists := ps.state[offset]
		if !exists {
			fmt.Printf("does not exit, off = %d, state = %v\n", offset, ps.state)
			continue
		}
		if msgState != MsgState_Pending {
			delete(ps.state, offset)
			logrus.WithFields(logrus.Fields{
				"OFFSET": offset,
				"PRTN":   ps.ID,
			}).Info("Removed offset")
			continue
		}
		latestToCommit.Offset = offset
		break
	}
	if latestToCommit.Offset == ps.lastCommited {
		return nil, fmt.Errorf("lastestToCommit is the same -> skipping")
	}
	return &latestToCommit, nil
}
