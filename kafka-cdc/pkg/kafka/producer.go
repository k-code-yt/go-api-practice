package pkgkafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
						"TOPIC_PRTN": ev.TopicPartition,
					}).Info("Delivery failed")
				} else {
					logrus.WithFields(logrus.Fields{
						"TOPIC_PRTN": ev.TopicPartition,
					}).Info("Delivery success")
				}
			}
		}
	}()

	return &KafkaProducer{
		producer: p,
	}
}

func (p *KafkaProducer) Produce(msg []byte) error {
	cfg := NewKafkaConfig()
	topics := cfg.DefaultTopics
	var err error

	for _, topic := range topics {
		err = p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          msg,
		}, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
