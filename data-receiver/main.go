package main

import (
	"encoding/json"
	"fmt"
	"go-logistic-api/shared"
	"log"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
)

type DataReceiver struct {
	conn      *websocket.Conn
	dataCH    chan []byte
	msgBroker *MsgBroker
}

func NewDataReceiver(conn *websocket.Conn, msgBroker *MsgBroker) *DataReceiver {
	return &DataReceiver{
		conn:      conn,
		dataCH:    make(chan []byte),
		msgBroker: msgBroker,
	}

}

type MsgBroker struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	readyCH  chan struct{}
}

func NewMsgBroker() (*MsgBroker, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": shared.Kafka_DefaultHost})
	if err != nil {
		return nil, err
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": shared.Kafka_DefaultHost,
		"group.id":          shared.Kafka_DefaultConsumerGroup,
		"auto.offset.reset": "beginning",
		// commit config
		"enable.auto.commit":      true,
		"auto.commit.interval.ms": 1000,

		// Reduce delays
		"fetch.min.bytes":       1,
		"fetch.wait.max.ms":     1000,
		"session.timeout.ms":    10000,
		"heartbeat.interval.ms": 3000,

		// Debug (remove in production)
		// "debug": "consumer,cgrp,topic,fetch",
	})

	if err != nil {
		return nil, err
	}

	err = c.SubscribeTopics([]string{shared.Kafka_DefaultTopic}, nil)

	if err != nil {
		return nil, err
	}

	return &MsgBroker{
		producer: p,
		consumer: c,
		readyCH:  make(chan struct{}),
	}, nil

}

func wsHandler(msgBroker *MsgBroker) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		wsID := r.Header.Get("Sec-Websocket-Key")
		if wsID != "" {
			fmt.Println("Received new WS conn = ", wsID)
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:    1028,
			WriteBufferSize:   1028,
			EnableCompression: true,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Fatal(err)
			return
		}

		dr := NewDataReceiver(conn, msgBroker)
		dr.acceptWSLoop()
	}
}

func (dr *DataReceiver) acceptWSLoop() {
	defer dr.conn.Close()
	fmt.Println("----STARTING TO ACCEPT")

	for {
		_, msgBytes, err := dr.conn.ReadMessage()
		if err != nil {
			fmt.Println("error receiving from WS conn ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				continue
			}
			break
		}

		go dr.msgBroker.produce(msgBytes)
	}
}

func (mb *MsgBroker) produce(b []byte) {
	// TODO -> remove later-on
	// what are the use-cases? pubsub? at-least once?
	go func() {
		for e := range mb.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
					continue
				}

				sensor := shared.SensorData{}
				err := json.Unmarshal(ev.Value, &sensor)
				if err != nil {
					fmt.Println("PRODUCER:err unmarshaling sensor data", err)
				}
				fmt.Printf("PRODUCER:Delivered message to %v\ndata = %+v\n", ev.TopicPartition, sensor)
			}
		}
	}()

	err := mb.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &shared.Kafka_DefaultTopic, Partition: kafka.PartitionAny},
		Value:          b,
	}, nil)
	if err != nil {
		fmt.Println("PRODUCER: error producing ", err)
		return
	}

}

func (mb *MsgBroker) consume() {
	defer mb.producer.Close()
	for {
		msg, err := mb.consumer.ReadMessage(time.Second)
		if err == nil {
			fmt.Printf("CONSUMER:Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else if !err.(kafka.Error).IsTimeout() {
			// The client will automatically try to recover from all errors.
			// Timeout is not considered an error because it is raised by
			// ReadMessage in absence of messages.
			fmt.Printf("CONSUMER: error %v (%v)\n", err, msg)
		}
	}

}

func (mb *MsgBroker) checkReadiness() <-chan struct{} {
	go func() {
		defer close(mb.readyCH)
		for {
			time.Sleep(1000 * time.Millisecond)
			fmt.Println("Consumer:RUNNING READY CHECK")
			assignment, err := mb.consumer.Assignment()
			if err != nil {
				log.Printf("Failed to get assignment: %v", err)
			}

			if len(assignment) > 0 {
				log.Println("Consumer: READY TO ACCEPT")
				return
			}
		}
	}()
	return mb.readyCH
}

func main() {
	msgBroker, err := NewMsgBroker()
	if err != nil {
		panic(err)
	}
	go msgBroker.consume()
	<-msgBroker.checkReadiness()

	http.HandleFunc("/", wsHandler(msgBroker))
	fmt.Printf("starting WS receiver on port %s\n", shared.WSPort)
	log.Fatal(http.ListenAndServe(shared.WSPort, nil))
}
