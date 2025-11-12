package withloopperclient

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
)

var (
	Kafka_DefaultTopic = "ws_chat"
)

type DataProducer interface {
	ProduceData(ReqMsg) error
}

type MsgProducer struct {
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

func (kp *KafkaProducer) ProduceData(data ReqMsg) error {
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("PRODUCER:Error marshaling %v\n", err)
		return err
	}

	err = kp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &Kafka_DefaultTopic, Partition: kafka.PartitionAny},
		Value:          b,
	}, nil)
	if err != nil {
		fmt.Println("PRODUCER: error producing ", err)
		return err
	}

	go func() {
		for e := range kp.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logrus.Infof("Delivery failed: %v\n", ev.TopicPartition)
					continue
				}

				msg := &ReqMsg{}
				err := json.Unmarshal(ev.Value, msg)
				if err != nil {
					logrus.Println("PRODUCER:err unmarshaling data", err)
				}
				logrus.Infof("PRODUCER:Delivered message to %v\ndata = %+v\n", ev.TopicPartition, msg)
			}
		}
	}()

	return nil

}

func NewMsgProducer() (*MsgProducer, error) {
	p, err := NewKafkaProducer()
	if err != nil {
		return nil, err
	}

	return &MsgProducer{
		producer: p,
	}, nil
}
