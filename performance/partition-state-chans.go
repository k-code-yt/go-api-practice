package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type ActionType string

const (
	ActionType_Add                 = "actiontype_add"
	ActionType_Update              = "actiontype_update"
	ActionType_Delete              = "actiontype_delete"
	ActionType_FindLastestToCommit = "actiontype_findlatesttocommit"
)

type UpdateStateMsg struct {
	toMsgState MsgState
	idx        int64
	actionType ActionType
}

func NewUpdateStateMsg(idx int64, actionType ActionType, toMsgState MsgState) *UpdateStateMsg {
	return &UpdateStateMsg{
		idx:        idx,
		toMsgState: toMsgState,
		actionType: actionType,
	}
}

type PartitionStateChans struct {
	State         map[int64]MsgState
	updateStateCH chan *UpdateStateMsg
	findResultCH  chan int64

	MaxReceived *atomic.Int64

	ctx    context.Context
	Cancel context.CancelFunc
	ExitCH chan struct{}
	wg     *sync.WaitGroup

	cfg *TestConfig
}

func NewPartitionStateChans(cfg *TestConfig) *PartitionStateChans {
	ctx, Cancel := context.WithCancel(context.Background())

	ps := &PartitionStateChans{
		State:         map[int64]MsgState{},
		updateStateCH: make(chan *UpdateStateMsg, 128),
		findResultCH:  make(chan int64, 64),

		MaxReceived: new(atomic.Int64),
		ctx:         ctx,
		Cancel:      Cancel,
		ExitCH:      make(chan struct{}),
		wg:          new(sync.WaitGroup),
		cfg:         cfg,
	}

	if cfg.prefillState > 0 {
		for idx := range ps.cfg.prefillState {
			ps.State[idx] = MsgState_Pending
		}
	}

	return ps
}

func (ps *PartitionStateChans) init() {
	ps.wg.Add(1 + 3*ps.cfg.numG)
	go func() {
		ps.wg.Wait()
		close(ps.updateStateCH)
		close(ps.ExitCH)
	}()

	go ps.updateStateLoop()

	for range ps.cfg.numG {
		go ps.appendLoop()
		go ps.commitLoop()
		go ps.updateLoop()
	}
}

func (ps *PartitionStateChans) cancel() {
	ps.Cancel()
}

func (ps *PartitionStateChans) exit() <-chan struct{} {
	return ps.ExitCH
}

func (ps *PartitionStateChans) updateStateLoop() {
	defer func() {
		ps.wg.Done()
		if ps.cfg.isDebugMode {
			logrus.WithFields(
				logrus.Fields{
					"IDX":      ps.cfg.IDX,
					"Scenario": ps.cfg.scenario,
				},
			).Info("EXIT updateStateLoop✅")
		}
	}()
	for {
		select {
		case <-ps.ctx.Done():
			return
		case msg := <-ps.updateStateCH:
			switch msg.actionType {
			case ActionType_Add, ActionType_Update:
				ps.State[msg.idx] = msg.toMsgState
			case ActionType_Delete:
				delete(ps.State, msg.idx)
			case ActionType_FindLastestToCommit:
				idx := ps.latestToCommitFinder()
				ps.findResultCH <- idx
			}
		}
	}
}

func (ps *PartitionStateChans) appendLoop() {
	t := time.NewTicker(ps.cfg.appendDur)
	defer func() {
		t.Stop()
		ps.wg.Done()
		if ps.cfg.isDebugMode {
			logrus.WithFields(
				logrus.Fields{
					"IDX":      ps.cfg.IDX,
					"Scenario": ps.cfg.scenario,
				},
			).Info("EXIT updateLoop✅")
		}
	}()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-t.C:
			// TODO -> add appender interface for different use-cases
			latest := ps.MaxReceived.Load()
			next := int64(0)

			if latest != 0 {
				next = latest + 1
				ps.MaxReceived.Store(next)
			} else {
				ps.MaxReceived.Store(1)
			}

			msg := NewUpdateStateMsg(next, ActionType_Add, MsgState_Pending)
			ps.updateStateCH <- msg
		}
	}
}

func (ps *PartitionStateChans) updateLoop() {
	t := time.NewTicker(ps.cfg.updateDur)

	defer func() {
		t.Stop()
		ps.wg.Done()
		if ps.cfg.isDebugMode {
			logrus.WithFields(
				logrus.Fields{
					"MaxReceived": ps.MaxReceived,
				},
			).Info("EXIT updateLoop✅")
		}
	}()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-t.C:
			// TODO -> add updater interface for different use-cases
			init := ps.cfg.InitOffset.Load()
			maxReceived := ps.MaxReceived.Load()
			if maxReceived == 0 {
				continue
			}

			maxToUpdate := math.Min(float64(init+ps.cfg.UpdateRange), float64(maxReceived))
			randInt := rand.Int64N(int64(maxToUpdate))
			if randInt < init {
				randInt += init
			}

			msg := NewUpdateStateMsg(randInt, ActionType_Add, MsgState_Success)
			ps.updateStateCH <- msg

			if ps.cfg.isDebugMode {
				fmt.Printf("Updated state for off = %d, init = %d, maxReceived = %d\n", randInt, init, maxReceived)
			}
		}
	}
}

func (ps *PartitionStateChans) commitLoop() {
	t := time.NewTicker(ps.cfg.commitDur)
	defer func() {
		t.Stop()
		ps.wg.Done()
		if ps.cfg.isDebugMode {
			logrus.WithFields(
				logrus.Fields{
					"MaxReceived": ps.MaxReceived,
				},
			).Info("EXIT commitOffsetLoop✅")
		}
	}()

	for {
		select {
		case <-t.C:
			select {
			case <-ps.ctx.Done():
				return
			default:
			}

			latestToCommit := ps.FindLatestToCommit()

			ps.cfg.InitOffset.Store(latestToCommit)

			if ps.cfg.isDebugMode {
				logrus.WithFields(
					logrus.Fields{
						"COMMITED_OFFSET": latestToCommit,
						"MAX_RECEIVED":    ps.MaxReceived.Load(),
					},
				).Warn("Commited on CRON")
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionStateChans) FindLatestToCommit() int64 {
	msg := NewUpdateStateMsg(-1, ActionType_FindLastestToCommit, MsgState_Success)
	ps.updateStateCH <- msg
	idx := <-ps.findResultCH
	return idx
}

func (ps *PartitionStateChans) latestToCommitFinder() int64 {
	latestToCommit := ps.MaxReceived.Load()
	initOffset := ps.cfg.InitOffset.Load()

	for offset := initOffset; offset <= latestToCommit; offset++ {
		msgState, exists := ps.State[offset]
		if !exists {
			if ps.cfg.isDebugMode {
				fmt.Printf("does not exit, off = %d, State = %v\n", offset, ps.State)
			}
			continue
		}

		if msgState != MsgState_Pending {
			delete(ps.State, offset)
			if ps.cfg.isDebugMode {
				logrus.WithFields(logrus.Fields{
					"OFFSET": offset,
				}).Info("Removed offset")
			}

			if len(ps.State) == 0 {
				latestToCommit = offset + 1
				break
			}
			continue
		}
		latestToCommit = offset
		break
	}

	if ps.cfg.isDebugMode {
		logrus.WithFields(logrus.Fields{
			"OFFSET": latestToCommit,
		}).Info("Offset to commit")
	}
	return latestToCommit
}
