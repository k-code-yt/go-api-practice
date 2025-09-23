package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-logistic-api/shared"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

type DataProducer interface {
	ProduceData(shared.SensorData) error
}

type DataConsumer interface {
	ConsumeData() error
	ReadyCheck() (bool, error)
}

type MsgBroker struct {
	producer DataProducer
	consumer DataConsumer
	readyCH  chan struct{}
}

type KafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer() (DataProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": shared.Kafka_DefaultHost})

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
	// go func() {
	// 	for e := range kp.producer.Events() {
	// 		switch ev := e.(type) {
	// 		case *kafka.Message:
	// 			if ev.TopicPartition.Error != nil {
	// 				fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
	// 				continue
	// 			}

	// 			sensor := shared.SensorData{}
	// 			err := json.Unmarshal(ev.Value, &sensor)
	// 			if err != nil {
	// 				fmt.Println("PRODUCER:err unmarshaling sensor data", err)
	// 			}
	// 			fmt.Printf("PRODUCER:Delivered message to %v\ndata = %+v\n", ev.TopicPartition, sensor)
	// 		}
	// 	}
	// }()

	return nil

}

type KafkaConsumer struct {
	consumer *kafka.Consumer
}

func NewKafkaConsumer() (DataConsumer, error) {

	err := initializeKafkaTopic(shared.Kafka_DefaultHost, shared.Kafka_DefaultTopic)
	if err != nil {
		fmt.Println("error creating topic")
		return nil, err
	}
	err = waitForTopicReady(shared.Kafka_DefaultHost, shared.Kafka_DefaultTopic)
	if err != nil {
		fmt.Println("error on topic created state")
		return nil, err
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        shared.Kafka_DefaultHost,
		"group.id":                 shared.Kafka_DefaultConsumerGroup,
		"allow.auto.create.topics": true,
		"auto.offset.reset":        "beginning",

		// commit config
		"enable.auto.commit": true,

		// Reduce delays
		"heartbeat.interval.ms": 3000,

		// Debug
		// "debug": "consumer,cgrp,topic,fetch",
	})

	if err != nil {
		return nil, err
	}

	err = c.SubscribeTopics([]string{shared.Kafka_DefaultTopic}, nil)

	if err != nil {
		return nil, err
	}
	return &KafkaConsumer{
		consumer: c,
	}, nil
}

func initializeKafkaTopic(brokers, topicName string) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	metadata, err := adminClient.GetMetadata(&topicName, false, 5000)
	if err == nil {
		if _, exists := metadata.Topics[topicName]; exists {
			logrus.Infof("Topic '%s' already exists", topicName)
			return nil
		}
	}

	// Create topic
	log.Printf("Creating topic '%s'...", topicName)
	topicSpec := kafka.TopicSpecification{
		Topic:             topicName,
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpec})
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("failed to create topic: %v", result.Error)
		}
		log.Printf("Topic '%s' created successfully", result.Topic)
	}

	return nil
}

func waitForTopicReady(brokers, topicName string) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	for {
		time.Sleep(1 * time.Second)
		metadata, err := adminClient.GetMetadata(&topicName, false, 5000)

		if err != nil {
			logrus.Errorf("Metadata fetch failed %v\n", err)
			continue
		}

		topicMeta, exists := metadata.Topics[topicName]
		if !exists {
			continue
		}

		if len(topicMeta.Partitions) > 0 {
			allPartitionsReady := true
			for _, partition := range topicMeta.Partitions {
				if partition.Error.Code() != kafka.ErrNoError {
					allPartitionsReady = false
					break
				}
				if partition.Leader == -1 {
					allPartitionsReady = false
					break
				}
			}

			logrus.WithField("IS_INITIALIZED", allPartitionsReady).Info("Cosumer Topic")

			if allPartitionsReady {
				return nil
			}
		}

	}

}

// func (kc *KafkaConsumer) ConsumeData(shared.SensorData) error {
func (kc *KafkaConsumer) ConsumeData() error {
	// TODO -> when to close producer? && consumer
	// defer kc.producer.Close()
	for {
		msg, err := kc.consumer.ReadMessage(time.Second)
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"partn": msg.TopicPartition,
				"val":   string(msg.Value),
			}).Info("CONSUMER:Message")
		} else if !err.(kafka.Error).IsTimeout() {
			// The client will automatically try to recover from all errors.
			// Timeout is not considered an error because it is raised by
			// ReadMessage in absence of messages.
			logrus.Errorf("CONSUMER: error %v (%v)\n", err, msg)
		}
	}
}
func (kc *KafkaConsumer) ReadyCheck() (bool, error) {
	assignment, err := kc.consumer.Assignment()
	if err != nil {
		logrus.Errorf("Failed to get assignment: %v", err)
		return false, err
	}

	return len(assignment) > 0, nil
}

func NewMsgBroker() (*MsgBroker, error) {
	p, err := NewKafkaProducer()
	if err != nil {
		return nil, err
	}
	p = NewLogMiddleware(p)

	c, err := NewKafkaConsumer()
	if err != nil {
		return nil, err
	}

	return &MsgBroker{
		producer: p,
		consumer: c,
		readyCH:  make(chan struct{}),
	}, nil

}

func (mb *MsgBroker) checkReadiness() <-chan struct{} {
	go func() {
		defer close(mb.readyCH)
		for {
			time.Sleep(3 * time.Second)
			isReady, err := mb.consumer.ReadyCheck()
			if err != nil {
				logrus.Error("Error on consumer readycheck")
				return
			}
			logrus.WithField("STATUS", isReady).Info("Consumer ready to accept")

			if isReady {
				return
			}
		}
	}()
	return mb.readyCH
}
