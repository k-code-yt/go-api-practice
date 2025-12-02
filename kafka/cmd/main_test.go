package main

import (
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/k-code-yt/go-api-practice/kafka/internal/consumer"
)

func BenchmarkPS_UpdateState(b *testing.B) {
	topic := "test_topic"

	cm := consumer.KafkaConsumer{}
	for i := 0; b.Loop(); i++ {
		offset := kafka.Offset(i)
		tp := &kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    offset,
		}

		cm.UpdateState(tp, consumer.MsgState_Pending)
	}
}

func BenchmarkPS_FindLatestToCommit(b *testing.B) {
	topic := "test_topic"

	tp := &kafka.TopicPartition{
		Topic:     &topic,
		Partition: 0,
		Offset:    0,
	}

	cm := consumer.NewTestKafkaConsumer(topic, tp)

	for i := 0; b.Loop(); i++ {
		offset := kafka.Offset(i)
		tp.Offset = offset
		if i%3 == 0 {
			cm.UpdateState(tp, consumer.MsgState_Pending)
			continue
		}

		cm.UpdateState(tp, consumer.MsgState_Success)
	}

	ps, err := cm.GetPartitionState(0)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = ps.FindLatestToCommit()
	}
}
