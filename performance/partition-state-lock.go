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

type PartitionStateLock struct {
	Mu    *sync.RWMutex
	State map[int64]MsgState

	MaxReceived *atomic.Int64

	ctx    context.Context
	Cancel context.CancelFunc
	ExitCH chan struct{}
	wg     *sync.WaitGroup

	cfg *TestConfig
}

func NewPartitionStateLock(cfg *TestConfig) *PartitionStateLock {
	ctx, Cancel := context.WithCancel(context.Background())
	ps := &PartitionStateLock{
		Mu:          &sync.RWMutex{},
		State:       map[int64]MsgState{},
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

func (ps *PartitionStateLock) init() {
	ps.wg.Add(3 * ps.cfg.numG)
	go func() {
		ps.wg.Wait()
		close(ps.ExitCH)
	}()

	for range ps.cfg.numG {
		go ps.appendLoop()
		go ps.commitLoop()
		go ps.updateLoop()
	}
}

func (ps *PartitionStateLock) cancel() {
	ps.Cancel()
}

func (ps *PartitionStateLock) exit() <-chan struct{} {
	return ps.ExitCH
}

func (ps *PartitionStateLock) appendLoop() {
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

			ps.Mu.Lock()
			ps.State[next] = MsgState_Pending
			ps.Mu.Unlock()
		}
	}
}

func (ps *PartitionStateLock) updateLoop() {
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
					},
				).Warn("Commited on CRON")
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionStateLock) FindLatestToCommit() (int64, error) {
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
