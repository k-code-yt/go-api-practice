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

type MsgState = int32

const (
	MsgState_Pending MsgState = iota
	MsgState_Success MsgState = iota
	MsgState_Error   MsgState = iota
)

type TestConfig struct {
	InitOffset  *atomic.Int64
	UpdateRange int64

	commitDur time.Duration
	updateDur time.Duration
	appendDur time.Duration

	isDebugMode bool
}

// commit, update, append durations
// will be used with milliseconds
func NewTestConfig(c, u, a int64, updateRange int64, isDebugMode bool) *TestConfig {
	init := new(atomic.Int64)
	init.Store(0)
	return &TestConfig{
		InitOffset:  init,
		UpdateRange: updateRange,

		commitDur: time.Millisecond * time.Duration(c),
		updateDur: time.Millisecond * time.Duration(u),
		appendDur: time.Millisecond * time.Duration(a),

		isDebugMode: isDebugMode,
	}
}

type PartitionStateLock struct {
	Mu    *sync.RWMutex
	State map[int64]MsgState

	MaxReceived *atomic.Int64

	ctx    context.Context
	Cancel context.CancelFunc
	ExitCH chan struct{}

	cfg *TestConfig
}

func NewPartitionStateLock(cfg *TestConfig) *PartitionStateLock {
	ctx, Cancel := context.WithCancel(context.Background())

	state := &PartitionStateLock{
		Mu:          &sync.RWMutex{},
		State:       map[int64]MsgState{},
		MaxReceived: new(atomic.Int64),
		ctx:         ctx,
		Cancel:      Cancel,
		ExitCH:      make(chan struct{}),
		cfg:         cfg,
	}
	return state
}

func (ps *PartitionStateLock) init() {
	go ps.appendLoop()
	go ps.commitLoop()
	go ps.updateLoop()

}

func (ps *PartitionStateLock) appendLoop() {
	t := time.NewTicker(ps.cfg.appendDur)

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

			ps.Mu.Lock()
			ps.State[next] = MsgState_Pending
			ps.Mu.Unlock()
		}
	}
}

func (ps *PartitionStateLock) updateLoop() {
	t := time.NewTicker(ps.cfg.updateDur)

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

			ps.Mu.Lock()
			ps.State[randInt] = MsgState_Success
			ps.Mu.Unlock()

			if ps.cfg.isDebugMode {
				fmt.Printf("Updated state for off = %d, init = %d, maxReceived = %d\n", randInt, init, maxReceived)
			}
		}
	}
}

func (ps *PartitionStateLock) commitLoop() {
	t := time.NewTicker(ps.cfg.commitDur)
	defer func() {
		close(ps.ExitCH)
		t.Stop()
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

			latestToCommit, err := ps.FindLatestToCommit()
			if err != nil {
				fmt.Println(err)
				continue
			}
			// TODO -> check if -1 here?
			ps.cfg.InitOffset.Store(latestToCommit)

			if ps.cfg.isDebugMode {
				logrus.WithFields(
					logrus.Fields{
						"COMMITED_OFFSET": latestToCommit,
						"MAX_RECEIVED":    ps.MaxReceived.Load(),
						"STATE":           ps.State,
					},
				).Warn("Commited on CRON")
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionStateLock) FindLatestToCommit() (int64, error) {
	if ps.cfg.isDebugMode {
		fmt.Printf("STATE = %+v\n", ps.State)
	}

	latestToCommit := ps.MaxReceived.Load()
	initOffset := ps.cfg.InitOffset.Load()
	// if initOffset == latestToCommit {
	// 	msg := fmt.Sprintf("lastCommit %d == MaxReceived -> skipping\n", initOffset)
	// 	return 0, fmt.Errorf("%v", msg)
	// }

	ps.Mu.Lock()
	defer ps.Mu.Unlock()
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
	return latestToCommit, nil
}

// TODO -> add interface
func (ps *PartitionStateLock) ReadOffset(offset int64) (MsgState, bool) {
	ps.Mu.RLock()
	defer ps.Mu.RUnlock()

	state, exists := ps.State[offset]
	return state, exists
}
