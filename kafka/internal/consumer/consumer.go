package consumer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/shared"
	"github.com/sirupsen/logrus"
)

type MsgState = int32

const (
	MsgState_Pending MsgState = iota
	MsgState_Success MsgState = iota
	MsgState_Error   MsgState = iota
)

type PartitionState struct {
	state        map[kafka.Offset]MsgState
	maxReceived  *kafka.TopicPartition
	mu           *sync.RWMutex
	lastCommited kafka.Offset
	exitCH       chan struct{}
}

func NewPartitionState(maxReceived *kafka.TopicPartition) *PartitionState {
	return &PartitionState{
		state:        map[kafka.Offset]MsgState{},
		maxReceived:  maxReceived,
		lastCommited: maxReceived.Offset,
		exitCH:       make(chan struct{}),
		mu:           &sync.RWMutex{},
	}

}

func (ps *PartitionState) commitOffsetLoop(commitDur time.Duration, c *KafkaConsumer) {
	ticker := time.NewTicker(commitDur)
	defer ticker.Stop()
	defer fmt.Printf("exiting PRNT State = %v\n", ps)
	for {
		select {
		case <-ticker.C:
			latestToCommit, err := ps.findLatestToCommit()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = c.consumer.CommitOffsets([]kafka.TopicPartition{*latestToCommit})
			if err != nil {
				fmt.Printf("err commiting offset = %d, prtn = %d, err = %v\n", latestToCommit.Offset, ps.maxReceived.Partition, err)
				continue
			}

			ps.mu.Lock()
			ps.lastCommited = latestToCommit.Offset - 1
			fmt.Printf("-----------state AFTER commit-----------\n")
			for offset, v := range ps.state {
				logrus.WithFields(
					logrus.Fields{
						"OFFSET": offset,
						"PRTN":   ps.maxReceived.Partition,
						"STATE":  v,
					},
				).Infof("STATE")
			}
			ps.mu.Unlock()

			logrus.WithFields(
				logrus.Fields{
					"OFFSET": latestToCommit.Offset - 1,
					"PRTN":   ps.maxReceived.Partition,
				},
			).Warn("Commited on CRON")

		case <-ps.exitCH:
			logrus.WithField("partition", ps.maxReceived.Partition).Info("Exiting commitOffsetLoop")
			return
		}
	}
}

func (ps *PartitionState) findLatestToCommit() (*kafka.TopicPartition, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.maxReceived == nil {
		return nil, fmt.Errorf("maxRec is nil")
	}
	latestToCommit := *ps.maxReceived
	if ps.lastCommited > ps.maxReceived.Offset {
		panic("last commit above maxReceived")
	}

	if ps.lastCommited == ps.maxReceived.Offset {
		msg := fmt.Sprintf("lastCommit %d == maxReceived in prtn %d -> skipping\n", ps.lastCommited, ps.maxReceived.Partition)
		return nil, fmt.Errorf("%v", msg)
	}

	for offset := ps.lastCommited + 1; offset < ps.maxReceived.Offset; offset++ {
		msgState, exists := ps.state[offset]
		if !exists {
			fmt.Printf("does not exit, off = %d, state = %v\n", offset, ps.state)
			continue
		}
		if msgState != MsgState_Pending {
			delete(ps.state, offset)
			continue
		}
		latestToCommit.Offset = offset
		break
	}
	if latestToCommit.Offset == ps.lastCommited {
		return nil, fmt.Errorf("lastestToCommit is the same -> skipping")
	}
	return &latestToCommit, nil
}

type KafkaConsumer struct {
	consumer     *kafka.Consumer
	MsgCH        chan *shared.Message
	isReady      bool
	readyCH      chan struct{}
	exitCH       chan struct{}
	topic        string
	msgsStateMap map[int32]*PartitionState
	mu           *sync.RWMutex
	commitDur    time.Duration
}

func NewKafkaConsumer(msgCH chan *shared.Message) *KafkaConsumer {
	cfg := shared.NewKafkaConfig()
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":               cfg.Host,
		"group.id":                        cfg.ConsumerGroup,
		"enable.auto.commit":              false,
		"auto.offset.reset":               "earliest", // Start from beginning if no commit
		"go.application.rebalance.enable": true,
		"partition.assignment.strategy":   "roundrobin", // or "roundrobin" or "cooperative-sticky"
		// "debug":                           "consumer,cgrp,topic",
	})

	if err != nil {
		panic(err)
	}

	tp := kafka.TopicPartition{
		Topic: &cfg.DefaultTopic,
	}
	commited, err := c.Committed([]kafka.TopicPartition{tp}, int(time.Second)*5)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	latestComm := kafka.OffsetBeginning
	if commited[0].Offset != kafka.OffsetInvalid {
		latestComm = commited[len(commited)-1].Offset
	}

	logrus.WithField("OFFSET", latestComm).Info("starting POSITION")

	consumer := &KafkaConsumer{
		consumer:     c,
		MsgCH:        msgCH,
		readyCH:      make(chan struct{}),
		exitCH:       make(chan struct{}),
		isReady:      false,
		topic:        cfg.DefaultTopic,
		mu:           new(sync.RWMutex),
		commitDur:    5 * time.Second,
		msgsStateMap: map[int32]*PartitionState{},
	}

	consumer.initializeKafkaTopic(cfg.Host, consumer.topic)

	err = c.SubscribeTopics([]string{consumer.topic}, consumer.rebalanceCB)
	if err != nil {
		panic(err)
	}

	go consumer.checkReadyToAccept()
	go consumer.consumeLoop()
	return consumer
}

func (c *KafkaConsumer) UpdateState(tp *kafka.TopicPartition, newState MsgState) {
	logrus.WithField("OFFSET", tp.Offset).Info("UpdateState")
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO -> err here
	c.msgsStateMap[tp.Partition].state[tp.Offset] = newState
}

func (c *KafkaConsumer) assignPrntCB(ev *kafka.AssignedPartitions) error {
	logrus.Info("=== Partitions Assigned ===")

	c.mu.Lock()

	committed, err := c.consumer.Committed(ev.Partitions, int(time.Second)*5)
	if err != nil {
		logrus.Errorf("Failed to get committed offsets: %v", err)
		committed = ev.Partitions
	}

	for i, tp := range committed {
		startOffset := tp.Offset
		if startOffset < 0 {
			startOffset = kafka.OffsetBeginning
		}

		logrus.WithFields(logrus.Fields{
			"partition":   tp.Partition,
			"startOffset": startOffset,
		}).Info("✅ Assigned partition")

		tpCopy := kafka.TopicPartition{
			Topic:     tp.Topic,
			Partition: tp.Partition,
			Offset:    startOffset,
		}

		prtnState := NewPartitionState(&tpCopy)
		c.msgsStateMap[tp.Partition] = prtnState

		ev.Partitions[i].Offset = startOffset
		go prtnState.commitOffsetLoop(c.commitDur, c)
	}

	c.mu.Unlock()

	err = c.consumer.Assign(ev.Partitions)
	if err != nil {
		logrus.Errorf("Failed to assign partitions: %v", err)
		return err
	}
	logrus.WithFields(logrus.Fields{
		"count":      len(ev.Partitions),
		"partitions": c.formatPartitions(ev.Partitions),
	}).Info("Successfully assigned partitions")

	return nil
}

func (c *KafkaConsumer) revokePrtnCB(ev *kafka.RevokedPartitions) error {
	logrus.Info("=== Partitions Revoked ===")

	var toCommit []kafka.TopicPartition
	for _, tp := range ev.Partitions {
		logrus.WithField("partition", tp.Partition).Info("❌ Revoking partition")

		c.mu.RLock()
		partitionState, exists := c.msgsStateMap[tp.Partition]
		c.mu.RUnlock()
		if !exists {
			continue
		}

		latestToCommit, err := partitionState.findLatestToCommit()
		if err != nil {
			fmt.Println(err)
			continue
		}

		if latestToCommit.Offset >= 0 {
			toCommit = append(toCommit, kafka.TopicPartition{
				Topic:     tp.Topic,
				Partition: tp.Partition,
				Offset:    latestToCommit.Offset,
			})
		}

		close(c.msgsStateMap[tp.Partition].exitCH)

		c.mu.Lock()
		delete(c.msgsStateMap, tp.Partition)
		c.mu.Unlock()

	}

	if len(toCommit) > 0 {
		_, err := c.consumer.CommitOffsets(toCommit)
		if err != nil {
			logrus.Errorf("Failed to commit on revoke: %v", err)
		} else {
			for _, tp := range toCommit {
				logrus.WithFields(logrus.Fields{
					"partition": tp.Partition,
					"offset":    tp.Offset - 1,
				}).Info("✅ Committed before revoke")
			}
		}
	}

	err := c.consumer.Unassign()
	if err != nil {
		logrus.Errorf("Failed to unassign partitions: %v", err)
		return err
	}

	logrus.Infof("Successfully revoked %d partitions", len(ev.Partitions))
	return nil
}

func (c *KafkaConsumer) rebalanceCB(_ *kafka.Consumer, event kafka.Event) error {
	switch ev := event.(type) {
	case kafka.AssignedPartitions:
		err := c.assignPrntCB(&ev)
		if err != nil {
			return err
		}
	case kafka.RevokedPartitions:
		err := c.revokePrtnCB(&ev)
		if err != nil {
			return err
		}
	default:
		logrus.Warnf("Unexpected event type: %T", ev)
	}
	return nil
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
		select {
		case c.MsgCH <- msgRequest:
		case <-time.After(5 * time.Second):
			logrus.Errorf("MsgCH blocked for 10s, dropping message offset=%d partition=%d",
				msg.TopicPartition.Offset, msg.TopicPartition.Partition)
			c.UpdateState(&msg.TopicPartition, MsgState_Error)
		}
	}

}

func (c *KafkaConsumer) appendMsgState(tp *kafka.TopicPartition) {
	c.mu.RLock()
	prtnState := c.msgsStateMap[tp.Partition]
	c.mu.RUnlock()

	prtnState.mu.Lock()
	defer prtnState.mu.Unlock()
	prtnState.state[tp.Offset] = MsgState_Pending
	if prtnState.maxReceived == nil || prtnState.maxReceived.Offset < tp.Offset {
		prtnState.maxReceived = &kafka.TopicPartition{
			Topic:     tp.Topic,
			Partition: tp.Partition,
			Offset:    tp.Offset,
		}
	}
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
		Topic:         topicName,
		NumPartitions: 4,
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

func (c *KafkaConsumer) formatPartitions(partitions []kafka.TopicPartition) string {
	parts := make([]string, len(partitions))
	for i, p := range partitions {
		parts[i] = fmt.Sprintf("%d@%d", p.Partition, p.Offset)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
