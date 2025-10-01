package main

import (
	"encoding/json"
	"fmt"

	"github.com/k-code-yt/go-api-practice/shared"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

type DataProducer interface {
	ProduceData(shared.SensorData) error
}

type MsgBroker struct {
	producer DataProducer
}

type KafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer() (DataProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":        shared.Kafka_DefaultHost,
		"allow.auto.create.topics": true,
	})

	if err != nil {
		return nil, err
	}
	return &KafkaProducer{
		producer: p,
	}, nil
}

func (kp *KafkaProducer) ProduceData(data shared.SensorData) error {
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("PRODUCER:Error marshaling %v\n", err)
		return err
	}

	err = kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &shared.Kafka_DefaultTopic, Partition: kafka.PartitionAny},
		Value:          b,
	}, nil)
	if err != nil {
		fmt.Println("PRODUCER: error producing ", err)
		return err
	}

	// TODO -> remove || for dev purposes only(for now)
	go func() {
		for e := range kp.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logrus.Infof("Delivery failed: %v\n", ev.TopicPartition)
					continue
				}

				sensor := shared.SensorData{}
				err := json.Unmarshal(ev.Value, &sensor)
				if err != nil {
					logrus.Println("PRODUCER:err unmarshaling sensor data", err)
				}
				logrus.Infof("PRODUCER:Delivered message to %v\ndata = %+v\n", ev.TopicPartition, sensor)
			}
		}
	}()

	return nil

}

func NewMsgBroker() (*MsgBroker, error) {
	p, err := NewKafkaProducer()
	if err != nil {
		return nil, err
	}
	p = NewLogMiddleware(p)

	return &MsgBroker{
		producer: p,
	}, nil
}
