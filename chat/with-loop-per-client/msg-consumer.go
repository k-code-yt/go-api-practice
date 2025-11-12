package withloopperclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
)

type DataConsumer interface {
	ReadMessageLoop()
}

type MsgConsumer struct {
	consumer DataConsumer
	readyCH  chan struct{}
}

func NewMsgConsumer(eventCH chan<- *ReqMsg) (*MsgConsumer, error) {
	c, err := NewKafkaConsumer(eventCH)
	if err != nil {
		// TODO  -> update err handling
		log.Fatal(err)
	}
	return &MsgConsumer{
		consumer: c,
		readyCH:  make(chan struct{}),
	}, nil
}

type KafkaConsumer struct {
	consumer *kafka.Consumer
	IsReady  bool
	eventCH  chan<- *ReqMsg
	// eventBus AggregatorEventBus[*shared.SensorData]
}

func NewKafkaConsumer(eventCH chan<- *ReqMsg) (DataConsumer, error) {
	err := initializeKafkaTopic(shared.Kafka_DefaultHost, Kafka_DefaultTopic)
	if err != nil {
		logrus.Error("error creating topic")
		return nil, err
	}
	err = waitForTopicReady(shared.Kafka_DefaultHost, Kafka_DefaultTopic)
	if err != nil {
		fmt.Println("error on topic created state")
		return nil, err
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": shared.Kafka_DefaultHost,
		"group.id":          "local_cg",
		"auto.offset.reset": "latest",

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

	err = c.SubscribeTopics([]string{Kafka_DefaultTopic}, nil)

	if err != nil {
		return nil, err
	}
	consumer := &KafkaConsumer{
		consumer: c,
		IsReady:  false,
		eventCH:  eventCH,
	}

	go consumer.checkReadyToAccept()
	return consumer, nil
}

func getConsumerGroup() string {
	// TODO -> get pod name
	return "local_cg"
}

func initializeKafkaTopic(brokers, topicName string) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	log.Printf("Creating topic '%s'...", topicName)
	topicSpec := kafka.TopicSpecification{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpec})
	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Error.Code() == kafka.ErrTopicAlreadyExists {
			logrus.Infof("Topic create result: %v", result.Error)
			continue
		}
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

func (kc *KafkaConsumer) readyCheck() (bool, error) {
	assignment, err := kc.consumer.Assignment()
	if err != nil {
		logrus.Errorf("Failed to get assignment: %v", err)
		return false, err
	}

	return len(assignment) > 0, nil
}

func (kc *KafkaConsumer) checkReadyToAccept() error {
	defer func() {
		kc.IsReady = true
	}()
	for {
		time.Sleep(1 * time.Second)
		isReady, err := kc.readyCheck()
		if err != nil {
			logrus.Error("Error on consumer readycheck")
			return err
		}
		logrus.WithField("STATUS", isReady).Warn("Consumer ready to accept")

		if isReady {
			return nil
		}
	}
}

func (kc *KafkaConsumer) ReadMessageLoop() {
	defer func() {
		logrus.WithField("Status", "Exiting").Warn("KAKFA:CONSUMER")
		kc.consumer.Close()
	}()

	// for kc.eventBus.IsActive() {
	for {
		msg, err := kc.consumer.ReadMessage(time.Millisecond * 100)
		if err != nil {
			if !err.(kafka.Error).IsTimeout() {
				logrus.Error("CONSUMER:Error on read message", err)
			}
			continue
		}

		data := new(ReqMsg)
		err = json.Unmarshal(msg.Value, data)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"val": string(msg.Value),
			}).Error("CONSUMER:Error Unmarshalling")
			continue
		}

		// TODO -> add generic chan
		// de, err := kc.mapTopicToDomainEvent(*msg.TopicPartition.Topic)
		// TODO add err chan
		kc.eventCH <- data
	}
}

// func (kc *KafkaConsumer) mapTopicToDomainEvent(topic string) ([]shared.DomainEvent, error) {
// 	if topic == "test_cg" {
// 		return []shared.DomainEvent{shared.AggregateDistanceDomainEvent, shared.CalculatePaymentDomainEvent}, nil
// 	}

// 	return nil, fmt.Errorf("domain event for %s kafka topic does not exist", topic)
// }
