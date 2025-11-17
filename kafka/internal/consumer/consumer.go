package consumer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/shared"
	"github.com/sirupsen/logrus"
)

type KafkaMsgState struct {
	kafka.TopicPartition
	completed bool
	commited  bool
}

func NewKafkaMsgState(tp *kafka.TopicPartition) *KafkaMsgState {
	return &KafkaMsgState{
		TopicPartition: *tp,
		completed:      false,
		commited:       false,
	}
}

type KafkaConsumer struct {
	consumer     *kafka.Consumer
	MsgCH        chan *shared.Message
	isReady      bool
	readyCH      chan struct{}
	topic        string
	msgsStates   []*KafkaMsgState
	msgsStateMap map[kafka.Offset]*KafkaMsgState
	lastOffset   kafka.Offset
	mu           *sync.RWMutex
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

	consumer := &KafkaConsumer{
		consumer:     c,
		MsgCH:        make(chan *shared.Message, 64),
		readyCH:      make(chan struct{}),
		isReady:      false,
		topic:        cfg.DefaultTopic,
		mu:           new(sync.RWMutex),
		msgsStates:   []*KafkaMsgState{},
		msgsStateMap: map[kafka.Offset]*KafkaMsgState{},
	}

	consumer.initializeKafkaTopic(cfg.Host, consumer.topic)

	// err = c.SubscribeTopics([]string{consumer.topic}, nil)
	// consumer.seekToOffset(int32(0), kafka.Offset(30))
	err = c.Assign([]kafka.TopicPartition{
		{
			Topic:     &consumer.topic,
			Partition: 0,
			// Offset:    kafka.Offset(170),
			Offset: kafka.OffsetEnd,
		},
	})
	if err != nil {
		panic(err)
	}

	go consumer.checkReadyToAccept()
	go consumer.consumeLoop()
	return consumer
}

func (c *KafkaConsumer) CommitMsg(tp *kafka.TopicPartition) {
	// TODO -> check for race
	fmt.Printf("Finished DB -> starting COMMIT for OFFSET = %d\n", tp.Offset)

	c.mu.Lock()
	defer c.mu.Unlock()

	msgState := c.msgsStateMap[tp.Offset]
	msgState.completed = true
	msgState.commited = true

	latestToCommitTP := &kafka.TopicPartition{}
	latestIdx := &atomic.Int64{}
	latestIdx.Store(-1)
	for idx, s := range c.msgsStates {
		if s.commited && idx == len(c.msgsStates)-1 {
			latestToCommitTP = c.extractAtIdx(int64(idx), latestIdx, latestToCommitTP)
			break
		}

		if s.commited {
			continue
		}

		prevIdx := int64(idx - 1)
		latestToCommitTP = c.extractAtIdx(prevIdx, latestIdx, latestToCommitTP)

		break
	}

	if latestIdx.Load() <= 0 {
		fmt.Println("Nothing to commit yet")
		return
	}
	// TODO -> goroutine?
	c.consumer.CommitOffsets([]kafka.TopicPartition{*latestToCommitTP})

	splitIdx := latestIdx.Load() + 1
	toRemove := c.msgsStates[:splitIdx]
	c.msgsStates = c.msgsStates[splitIdx:]
	for _, s := range toRemove {
		delete(c.msgsStateMap, s.Offset)
	}
	firstToCheckMsgs := -1
	if len(c.msgsStates) > 0 {
		firstToCheck := c.msgsStates[0]
		firstToCheckMsgs = int(firstToCheck.Offset)
	}

	logrus.WithFields(
		logrus.Fields{
			"OFFSET":                 latestToCommitTP.Offset,
			"msgsStateLEN":           len(c.msgsStates),
			"firstToCheckMsgsOFFSET": firstToCheckMsgs,
			"lastRemovedOFFSET":      toRemove[len(toRemove)-1].Offset,
		},
	).Info("Committed after DB operation")
}

func (c *KafkaConsumer) extractAtIdx(idx int64, latestIdx *atomic.Int64, latestToCommitTP *kafka.TopicPartition) *kafka.TopicPartition {
	latestIdx.Store(int64(idx))
	if idx > 0 {
		latestToCommitTP = &c.msgsStates[idx].TopicPartition
		logrus.WithField("OFFSET", latestToCommitTP.Offset+1).Info("not ready OFFSET")
	}

	for _, s := range c.msgsStates {
		logrus.WithFields(
			logrus.Fields{
				"commited": s.commited,
				"OFFSET":   s.TopicPartition.Offset,
			},
		).Info("Before CMT State")
	}
	return latestToCommitTP
}

func (c *KafkaConsumer) consumeLoop() {
	defer c.consumer.Close()
	firstMsg := true

	for {
		msg, err := c.consumer.ReadMessage(time.Second)
		if err != nil && err.(kafka.Error).IsTimeout() {
			continue
		}
		if err != nil && !err.(kafka.Error).IsTimeout() {
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

		c.appendMsgState(&msg.TopicPartition)

		msgRequest := shared.NewMessage(&msg.TopicPartition, msg.Value)
		// fmt.Printf("received msg = %+v\n", msgRequest)
		c.MsgCH <- msgRequest
	}

}

func (c *KafkaConsumer) appendMsgState(tp *kafka.TopicPartition) {
	c.mu.Lock()
	msgState := NewKafkaMsgState(tp)
	c.msgsStates = append(c.msgsStates, msgState)
	c.msgsStateMap[tp.Offset] = msgState
	c.mu.Unlock()
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

func (c *KafkaConsumer) seekToOffset(partition int32, offset kafka.Offset) error {
	err := c.consumer.Seek(kafka.TopicPartition{
		Topic:     &c.topic,
		Partition: partition,
		Offset:    offset, // ← HERE
	}, 1000) // timeout in ms

	return err
}
