package consumer

import (
	"context"
	"fmt"
	"maps"
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

type PartitionState struct {
	ID                int32
	state             map[kafka.Offset]shared.MsgState
	maxReceived       *kafka.TopicPartition
	lastCommited      kafka.Offset
	ctx               context.Context
	cancel            context.CancelFunc
	exitCH            chan struct{}
	deleteFromStateCH chan kafka.Offset
	updateStateCH     chan *UpdateStateMsg
	getStateSizeCH    chan chan int
}

func NewPartitionState(maxReceived *kafka.TopicPartition) *PartitionState {
	ctx, cancel := context.WithCancel(context.Background())
	initialLastCommited := maxReceived.Offset - 1
	if maxReceived.Offset == kafka.OffsetBeginning || maxReceived.Offset < 0 {
		initialLastCommited = -1
	}
	ps := &PartitionState{
		ID:                maxReceived.Partition,
		state:             map[kafka.Offset]shared.MsgState{},
		maxReceived:       maxReceived,
		lastCommited:      initialLastCommited,
		ctx:               ctx,
		cancel:            cancel,
		exitCH:            make(chan struct{}),
		deleteFromStateCH: make(chan kafka.Offset, 128),
		updateStateCH:     make(chan *UpdateStateMsg, 128),
		getStateSizeCH:    make(chan chan int, 16),
	}
	go ps.acceptMsgLoop()
	return ps
}

func (ps *PartitionState) acceptMsgLoop() {
	for {
		select {
		case <-ps.ctx.Done():
			return
		case offset := <-ps.deleteFromStateCH:
			delete(ps.state, offset)
			logrus.WithFields(logrus.Fields{
				"OFFSET": offset,
				"PRTN":   ps.ID,
			}).Info("Removed offset")

		case msg := <-ps.updateStateCH:
			ps.state[msg.offset] = msg.value
		case respCH := <-ps.getStateSizeCH:
			respCH <- len(ps.state)
		}
	}

}
func (ps *PartitionState) commitOffsetLoop(commitDur time.Duration, c *KafkaConsumer) {
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

			stateCopy := maps.Clone(ps.state)
			latestToCommit, err := ps.findLatestToCommit(stateCopy)
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = c.consumer.CommitOffsets([]kafka.TopicPartition{*latestToCommit})
			if err != nil {
				fmt.Printf("err commiting offset = %d, prtn = %d, err = %v\n", latestToCommit.Offset, ps.maxReceived.Partition, err)
				continue
			}

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

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionState) findLatestToCommit(stateCopy map[kafka.Offset]shared.MsgState) (*kafka.TopicPartition, error) {
	defer func() {
		stateCopy = nil
	}()
	fmt.Printf("PRTN = %d, STATE = %+v\n", ps.ID, ps.state)

	if ps.maxReceived == nil {
		return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.maxReceived
	if ps.lastCommited > ps.maxReceived.Offset {
		fmt.Println("❌last commit above maxReceived❌")
		fmt.Printf("last = %d, max = %d\n", ps.lastCommited, ps.maxReceived.Offset)
		// panic("❌last commit above maxReceived❌")
	}
	if ps.lastCommited == ps.maxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == maxReceived in prtn %d -> skipping\n", ps.lastCommited, ps.maxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}
	for offset := ps.lastCommited; offset <= ps.maxReceived.Offset+1; offset++ {
		if ps.getStateSize() == 0 {
			latestToCommit.Offset = offset + 1
			break
		}

		msgState, exists := stateCopy[offset]
		if !exists {
			// fmt.Printf("does not exit, off = %d, state = %v\n", offset, ps.state)
			continue
		}
		if msgState != shared.MsgState_Pending {
			ps.deleteFromStateCH <- offset
			continue
		}
		latestToCommit.Offset = offset
		break
	}
	return &latestToCommit, nil
}

func (ps *PartitionState) getStateSize() int {
	respCH := make(chan int)
	defer close(respCH)
	ps.getStateSizeCH <- respCH
	return <-respCH
}
