package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	HttpPort = ":8080"
)

type TestReq struct {
	UpdateRange int64 `json:"update_range"`

	CommitDur int64 `json:"commit_ms"`
	UpdateDur int64 `json:"update_ms"`
	AppendDur int64 `json:"append_ms"`
}

type TestScenario string

const (
	TestScenario_Lock    = "lock"
	TestScenario_SyncMap = "syncmap"
	TestScenario_Chan    = "chan"
)

func testCaseHandler(w http.ResponseWriter, r *http.Request) {
	var body TestReq

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		errMsg := fmt.Sprintf("failer to parse body %v", err)
		sendError(w, errMsg)
		return
	}

	vars := mux.Vars(r)
	scenario, ok := vars["scenario"]
	if !ok {
		errMsg := fmt.Sprintln("scenario was not provided")
		sendError(w, errMsg)
		return
	}
	fmt.Println(scenario)

	cfg := NewTestConfig(
		body.CommitDur,
		body.UpdateDur,
		body.AppendDur,
		body.UpdateRange,
		true,
	)

	switch scenario {
	case TestScenario_Lock:
		ps := NewPartitionStateLock(cfg)
		ps.init()
	case TestScenario_SyncMap:
		ps := NewPartitionStateSyncMap(cfg)
		ps.init()
	}

	resp := map[string]bool{"received": true}
	b, err := json.Marshal(resp)

	w.Header().Set("status", strconv.Itoa(http.StatusOK))
	w.Write(b)
}

func sendError(w http.ResponseWriter, errMsg string) {
	fmt.Println(errMsg)
	w.Header().Set("status", strconv.Itoa(http.StatusInternalServerError))
	w.Write([]byte(errMsg))
}

func httpServer() {
	mux := mux.NewRouter()
	mux.HandleFunc("/test/{scenario}", testCaseHandler).Methods("POST")

	fmt.Printf("HTTP server is listening on %s\n", HttpPort)

	log.Fatal(http.ListenAndServe(HttpPort, mux))
}

func main() {
	startMetricsServer()
	httpServer()
}
