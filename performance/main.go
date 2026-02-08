package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	HttpPort = ":8080"
)

type StateRunner interface {
	init()
	cancel()
	exit() <-chan struct{}
}

type TestReq struct {
	UpdateRange int64 `json:"update_range"`

	CommitDur int64 `json:"commit_ms"`
	UpdateDur int64 `json:"update_ms"`
	AppendDur int64 `json:"append_ms"`

	TestDur  int64  `json:"test_ms"`
	Scenario string `json:"scenario"`
}

type TestScenario string

const (
	TestScenario_Lock    = "lock"
	TestScenario_SyncMap = "syncmap"
	TestScenario_Chan    = "chan"
)

type CurrentTest struct {
	cfgs   []*TestConfig
	ctx    context.Context
	cancel context.CancelFunc
	exitCH chan struct{}
}

func NewCurrentTest() *CurrentTest {
	ctx, cancel := context.WithCancel(context.Background())
	return &CurrentTest{
		ctx:    ctx,
		cancel: cancel,
		exitCH: make(chan struct{}),
	}
}

type Server struct {
	currentTest *CurrentTest
}

func (s *Server) testCaseHandler(w http.ResponseWriter, r *http.Request) {
	if s.currentTest != nil {
		s.currentTest.cancel()
		<-s.currentTest.exitCH
		s.currentTest = nil
	}
	s.currentTest = NewCurrentTest()

	var body []TestReq

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		errMsg := fmt.Sprintf("failer to parse body %v", err)
		sendError(w, errMsg)
		return
	}

	cfgs := make([]*TestConfig, len(body))
	for idx, item := range body {
		cfg := NewTestConfig(
			idx,
			item.CommitDur,
			item.UpdateDur,
			item.AppendDur,
			item.TestDur,
			item.UpdateRange,
			TestScenario(item.Scenario),
			1,
			-1,
			true,
		)

		cfgs[idx] = cfg
	}

	go s.runTests(cfgs)

	resp := map[string]bool{"received": true}
	b, err := json.Marshal(resp)

	w.Header().Set("status", strconv.Itoa(http.StatusOK))
	w.Write(b)
}

func (s *Server) runTests(cfgs []*TestConfig) {
	defer func() {
		close(s.currentTest.exitCH)
		logrus.Debug("CANCEL_PREV_TEST")
	}()
	for _, cfg := range cfgs {
		select {
		case <-s.currentTest.ctx.Done():
			return
		default:
			var ps StateRunner
			switch cfg.scenario {
			case TestScenario_Lock:
				ps = NewPartitionStateLock(cfg)
			case TestScenario_SyncMap:
				ps = NewPartitionStateSyncMap(cfg)
			}

			go func(ps StateRunner) {
				<-time.After(cfg.testDur)
				ps.cancel()
			}(ps)

			ps.init()
			<-ps.exit()

			// trigger my clean up here
			ps = nil
			// sleep for gc to cleanup
			time.Sleep(time.Second * 30)
		}
	}
}

func sendError(w http.ResponseWriter, errMsg string) {
	fmt.Println(errMsg)
	w.Header().Set("status", strconv.Itoa(http.StatusInternalServerError))
	w.Write([]byte(errMsg))
}

func httpServer() {
	mux := mux.NewRouter()
	s := Server{}
	mux.HandleFunc("/test", s.testCaseHandler).Methods("POST")

	fmt.Printf("HTTP server is listening on %s\n", HttpPort)

	log.Fatal(http.ListenAndServe(HttpPort, mux))
}

func main() {
	startMetricsServer()
	httpServer()
}
