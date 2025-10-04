package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	client "github.com/k-code-yt/go-api-practice/data-aggregator/transport"
	"github.com/k-code-yt/go-api-practice/shared"
	"github.com/sirupsen/logrus"
)

func handleGetDistance(svc Aggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		id := queryParams.Get("id")
		d := svc.GetDistance(id)
		err := writeJSON(w, http.StatusOK, d)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "request failed, unable to marshal payload"})
			return
		}
	}
}

func handleGetInvoice(intergation client.TransportClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		id := queryParams.Get("id")
		d, err := intergation.GetInvoice(id)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "request failed to fetch invoice"})
			return
		}
		err = writeJSON(w, http.StatusOK, d)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "request failed, unable to marshal payload"})
			return
		}
	}
}

func handleWS(dataCH chan *shared.SensorDataProto) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.Header.Get("Sec-Websocket-Key")
		if wsID != "" {
			logrus.Infof("Received new WS conn = %s", wsID)
		}

		upgrader := websocket.Upgrader{
			ReadBufferSize:  512,
			WriteBufferSize: 512,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logrus.Errorf("error establish ws conn %v", err)
			return
		}

		conn.SetReadDeadline(time.Time{})
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Time{})
			return nil
		})

		go func(conn *websocket.Conn) {
			defer conn.Close()
			for {
				_, b, err := conn.ReadMessage()
				if err != nil {
					logrus.Errorf("error reading ws conn %v", err)
					break
				}
				logrus.Infof("received msg via WS %s", string(b))
				sensorDataReq := new(shared.SensorData)
				json.Unmarshal(b, sensorDataReq)
				sensorDataMSG := new(shared.SensorDataProto)
				sensorDataMSG.ID = sensorDataReq.SensorID.String()
				sensorDataMSG.Lat = sensorDataReq.Lat
				sensorDataMSG.Lng = sensorDataReq.Lng
				dataCH <- sensorDataMSG
			}
		}(conn)
	}
}

func writeJSON(rw http.ResponseWriter, status int, v any) error {
	rw.WriteHeader(status)
	rw.Header().Add("Content-Type", "applicaiton/json")
	return json.NewEncoder(rw).Encode(v)
}
