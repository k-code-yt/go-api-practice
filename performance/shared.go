package main

import (
	"sync/atomic"
	"time"
)

type MsgState = int32

const (
	MsgState_Pending MsgState = iota
	MsgState_Success MsgState = iota
	MsgState_Error   MsgState = iota
)

type TestConfig struct {
	IDX         int
	InitOffset  *atomic.Int64
	UpdateRange int64
	numG        int

	commitDur time.Duration
	updateDur time.Duration
	appendDur time.Duration

	scenario     TestScenario
	testDur      time.Duration
	prefillState int64
	isDebugMode  bool
}

// commit, update, append, test durations
// will be used with milliseconds
// updateRange => how many items will be read for commitLoop
func NewTestConfig(
	idx int,
	commitdur,
	updatedur,
	appenddur int64,
	testdur int64,
	updateRange int64,
	scenario TestScenario,
	numG int,
	prefillState int64,
	isDebugMode bool,
) *TestConfig {
	init := new(atomic.Int64)
	init.Store(0)
	if numG <= 0 {
		numG = 1
	}

	return &TestConfig{
		IDX:         idx,
		InitOffset:  init,
		UpdateRange: updateRange,

		commitDur: time.Millisecond * time.Duration(commitdur),
		updateDur: time.Millisecond * time.Duration(updatedur),
		appendDur: time.Millisecond * time.Duration(appenddur),

		scenario:     scenario,
		testDur:      time.Millisecond * time.Duration(testdur),
		prefillState: prefillState,
		//TODO -> revert
		numG:        1,
		isDebugMode: isDebugMode,
	}
}
