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

type PartitionStateSyncMap struct {
	State *sync.Map

	MaxReceived *atomic.Int64

	ctx    context.Context
	Cancel context.CancelFunc
	ExitCH chan struct{}
	wg     *sync.WaitGroup

	cfg *TestConfig
}

func NewPartitionStateSyncMap(cfg *TestConfig) *PartitionStateSyncMap {
	ctx, Cancel := context.WithCancel(context.Background())

	state := &PartitionStateSyncMap{
		State:       new(sync.Map),
		MaxReceived: new(atomic.Int64),

		ctx:    ctx,
		Cancel: Cancel,
		wg:     new(sync.WaitGroup),
		ExitCH: make(chan struct{}),
		cfg:    cfg,
	}
	return state
}

func (ps *PartitionStateSyncMap) init() {
	ps.wg.Add(3)
	go func() {
		ps.wg.Wait()
		close(ps.ExitCH)
	}()
	go ps.appendLoop()
	go ps.commitLoop()
	go ps.updateLoop()
}

func (ps *PartitionStateSyncMap) cancel() {
	ps.Cancel()
}

func (ps *PartitionStateSyncMap) exit() <-chan struct{} {
	return ps.ExitCH
}

func (ps *PartitionStateSyncMap) appendLoop() {
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
			).Info("EXIT appendLoop✅")
		}
	}()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-t.C:
			latest := ps.MaxReceived.Load()
			next := int64(0)

			if latest != 0 {
				next = latest + 1
				ps.MaxReceived.Store(next)
			} else {
				ps.MaxReceived.Store(1)
			}

			ps.State.Store(next, MsgState_Pending)
		}
	}
}

func (ps *PartitionStateSyncMap) updateLoop() {
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

			ps.State.Swap(randInt, MsgState_Success)

			if ps.cfg.isDebugMode {
				fmt.Printf("Updated state for off = %d, init = %d, maxReceived = %d\n", randInt, init, maxReceived)
			}
		}
	}
}

func (ps *PartitionStateSyncMap) commitLoop() {
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
						"STATE":           ps.State,
					},
				).Warn("Commited on CRON")
			}

		case <-ps.ctx.Done():
			return
		}
	}
}

func (ps *PartitionStateSyncMap) FindLatestToCommit() (int64, error) {
	if ps.cfg.isDebugMode {
		fmt.Printf("STATE = %+v\n", ps.State)
	}

	latestToCommit := ps.MaxReceived.Load()
	initOffset := ps.cfg.InitOffset.Load()
	// if initOffset == latestToCommit {
	// 	msg := fmt.Sprintf("lastCommit %d == MaxReceived -> skipping\n", initOffset)
	// 	return 0, fmt.Errorf("%v", msg)
	// }

	for offset := initOffset; offset <= latestToCommit; offset++ {
		msgState, exists := ps.State.Load(offset)
		if !exists {
			if ps.cfg.isDebugMode {
				fmt.Printf("does not exit, off = %d, State = %v\n", offset, ps.State)
			}
			continue
		}

		if msgState != MsgState_Pending {
			ps.State.Delete(offset)
			if ps.cfg.isDebugMode {
				logrus.WithFields(logrus.Fields{
					"OFFSET": offset,
				}).Info("Removed offset")
			}

			if getSyncMapLen(ps.State) == 0 {
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

func getSyncMapLen(state *sync.Map) int {
	count := 0

	state.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	return count
}

// TODO -> add interface
func (ps *PartitionStateSyncMap) ReadOffset(offset int64) (MsgState, bool) {
	val, ok := ps.State.Load(offset)
	valstate := val.(MsgState)
	return valstate, ok
}
