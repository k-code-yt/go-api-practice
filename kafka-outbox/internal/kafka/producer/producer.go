package producer

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka-outbox/internal/config"
	"github.com/sirupsen/logrus"
)

type KafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer() *KafkaProducer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}

	go func() {
		defer p.Close()
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logrus.WithFields(logrus.Fields{
						"PRTN": ev.TopicPartition,
					}).Warn("Delivery failed")
				} else {
					logrus.WithFields(logrus.Fields{
						"PRTN":   ev.TopicPartition.Partition,
						"OFFSET": ev.TopicPartition.Offset,
					}).Warn("Delivery success")
				}
			}
		}
	}()

	return &KafkaProducer{
		producer: p,
	}
}

func (p *KafkaProducer) Produce(msg []byte) error {
	cfg := config.NewKafkaConfig()
	topic := cfg.DefaultTopic
	return p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          msg,
	}, nil)
}
