package main

import (
	"fmt"
	"go-logistic-api/shared"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	coordEmitDurr time.Duration = time.Millisecond * 1000
	truckCount    int           = 5000
)

func NewSensorData(id uuid.UUID) *shared.SensorData {
	lat, lng := getLatLng()
	return &shared.SensorData{
		SensorID: id,
		Lat:      lat,
		Lng:      lng,
	}
}

func getCoordinates() float64 {
	coord := float64(rand.Intn(100) + 1)
	remainder := rand.Float64()
	return coord + remainder
}

func getLatLng() (float64, float64) {
	return getCoordinates(), getCoordinates()
}

func getSensorIds(n int) []uuid.UUID {
	ids := []uuid.UUID{}

	for range n {
		id := uuid.New()
		ids = append(ids, id)
	}

	return ids
}

func main() {
	sensorIds := getSensorIds(truckCount)
	dialer := websocket.Dialer{
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
	}

	conn, resp, err := dialer.Dial(shared.WSEndpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	extension := resp.Header.Get("Sec-WebSocket-Extensions")
	if extension != "" {
		fmt.Println("Compression enabled")
	} else {
		fmt.Println("Compression was not negotiated with receiver")
	}

	for _, id := range sensorIds {
		sd := NewSensorData(id)
		fmt.Println("sending data = ", sd)
		err := conn.WriteJSON(sd)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(coordEmitDurr)
	}

}
