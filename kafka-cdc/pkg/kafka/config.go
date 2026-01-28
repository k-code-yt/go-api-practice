package pkgkafka

import (
	"fmt"
)

type KafkaConfig struct {
	DefaultTopics            []string
	TopicsCount              int
	Host                     string
	ConsumerGroup            string
	ParititionAssignStrategy string
	NumPartitions            int
	MsgEncoder               MsgEncoder
}

func NewKafkaConfig() *KafkaConfig {
	strategy := "cooperative-sticky"
	count := 1
	topics := []string{}
	for id := range count {
		t := fmt.Sprintf("%s_topic_%d", strategy, id)
		topics = append(topics, t)
	}

	cfg := &KafkaConfig{
		ParititionAssignStrategy: strategy,
		DefaultTopics:            topics,
		Host:                     "localhost",
		ConsumerGroup:            "local_cg",
		NumPartitions:            4,
	}
	return cfg
}
