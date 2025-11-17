package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/shared"
	"github.com/sirupsen/logrus"
)

type KafkaConsumer struct {
	consumer          *kafka.Consumer
	MsgCH             chan *shared.Message
	isReady           bool
	readyCH           chan struct{}
	topic             string
	offsesToCommitMap []*KafkaMsg
	lastOffset        kafka.Offset
	mu                *sync.Mutex
}

type KafkaMsg struct {
	kafka.TopicPartition
	completed bool
	commited  bool
}

func NewKafkaConsumer() *KafkaConsumer {
	cfg := shared.NewKafkaConfig()
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  cfg.Host,
		"group.id":           cfg.ConsumerGroup,
		"enable.auto.commit": false,
	})

	if err != nil {
		panic(err)
	}
	//enable.auto.commit

	consumer := &KafkaConsumer{
		consumer: c,
		MsgCH:    make(chan *shared.Message, 64),
		readyCH:  make(chan struct{}),
		isReady:  false,
		topic:    cfg.DefaultTopic,
	}

	consumer.initializeKafkaTopic(cfg.Host, consumer.topic)

	// err = c.SubscribeTopics([]string{consumer.topic}, nil)
	// consumer.seekToOffset(int32(0), kafka.Offset(30))
	err = c.Assign([]kafka.TopicPartition{
		{
			Topic:     &consumer.topic,
			Partition: 0,
			Offset:    kafka.Offset(170),
			// Offset: kafka.OffsetEnd,
		},
	})
	if err != nil {
		panic(err)
	}

	go consumer.checkReadyToAccept()
	go consumer.consumeLoop()
	return consumer
}

func (c *KafkaConsumer) seekToOffset(partition int32, offset kafka.Offset) error {
	err := c.consumer.Seek(kafka.TopicPartition{
		Topic:     &c.topic,
		Partition: partition,
		Offset:    offset, // ← HERE
	}, 1000) // timeout in ms

	return err
}

func (c *KafkaConsumer) CommitMsg(msg *kafka.TopicPartition) {
	time.Sleep(2 * time.Second)
	c.consumer.CommitOffsets([]kafka.TopicPartition{*msg})
	fmt.Printf("committed msg = %d\n", msg.Offset)
}

func (c *KafkaConsumer) consumeLoop() {
	defer c.consumer.Close()
	firstMsg := true

	for {
		msg, err := c.consumer.ReadMessage(time.Second)
		if err != nil && !err.(kafka.Error).IsTimeout() {
			// The client will automatically try to recover from all errors.
			// Timeout is not considered an error because it is raised by
			// ReadMessage in absence of messages.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
			continue
		}
		if msg == nil {
			continue
		}

		if firstMsg {
			close(c.readyCH)
		}

		firstMsg = false
		msgRequest := shared.NewMessage(&msg.TopicPartition, msg.Value)
		fmt.Printf("received msg = %+v", msgRequest)
		c.MsgCH <- msgRequest
	}

}

func (c *KafkaConsumer) checkReadyToAccept() error {
	defer func() {
		c.isReady = true
	}()
	for {
		select {
		case <-c.readyCH:
			return nil
		default:
			time.Sleep(1 * time.Second)
			isReady, err := c.readyCheck()
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
}

func (c *KafkaConsumer) readyCheck() (bool, error) {
	assignment, err := c.consumer.Assignment()
	if err != nil {
		logrus.Errorf("Failed to get assignment: %v", err)
		return false, err
	}

	return len(assignment) > 0, nil
}

func (c *KafkaConsumer) initializeKafkaTopic(brokers, topicName string) error {
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
			logrus.Infof("Topic already exists: %v", result.Error)
			continue
		}
		if result.Error.Code() != kafka.ErrNoError {
			return fmt.Errorf("failed to create topic: %v", result.Error)
		}
		log.Printf("Topic '%s' created successfully", result.Topic)
	}

	return c.waitForTopicReady(brokers, topicName)
}

func (c *KafkaConsumer) waitForTopicReady(brokers, topicName string) error {
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
